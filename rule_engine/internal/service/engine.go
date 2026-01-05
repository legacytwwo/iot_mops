package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"rule_engine/internal/entities"
	rstate "rule_engine/internal/state"
)

type RulesRepository interface {
	FindEnabledByMetric(ctx context.Context, metric string) ([]entities.Rule, error)
	FindInactivityRules(ctx context.Context) ([]entities.Rule, error)
}

type AlertsRepository interface {
	Insert(ctx context.Context, alert entities.Alert) error
	InsertMany(ctx context.Context, alerts []entities.Alert) error
}

type StateStore interface {
	GetRuleState(ctx context.Context, ruleID, metric, deviceID string) (rstate.RuleState, bool, error)
	SetRuleState(ctx context.Context, ruleID, metric, deviceID string, state rstate.RuleState, ttl time.Duration) error
	DeleteRuleState(ctx context.Context, ruleID, metric, deviceID string) error

	AddToTimeWindow(ctx context.Context, ruleID, metric, deviceID, member string, ts time.Time, ttl time.Duration) error
	PruneTimeWindow(ctx context.Context, ruleID, metric, deviceID string, from time.Time) error
	CountTimeWindow(ctx context.Context, ruleID, metric, deviceID string) (int64, error)
	DeleteTimeWindow(ctx context.Context, ruleID, metric, deviceID string) error

	GetLastSeen(ctx context.Context, deviceID string) (time.Time, bool, error)
	TryMarkMessage(ctx context.Context, messageID string, ttl time.Duration) (bool, error)
	DeleteMessageMark(ctx context.Context, messageID string) error
}

type RuleEngine struct {
	rulesRepo  RulesRepository
	alertsRepo AlertsRepository
	state      StateStore
	logger     *zap.Logger

	messageTTL time.Duration
}

func NewRuleEngine(rules RulesRepository, alerts AlertsRepository, state StateStore, logger *zap.Logger, messageTTL time.Duration) *RuleEngine {
	return &RuleEngine{
		rulesRepo:  rules,
		alertsRepo: alerts,
		state:      state,
		logger:     logger,
		messageTTL: messageTTL,
	}
}

func (e *RuleEngine) ProcessEnvelope(ctx context.Context, env entities.Envelope) ([]entities.Alert, error) {
	if env.MessageID != "" {
		ok, err := e.state.TryMarkMessage(ctx, env.MessageID, e.messageTTL)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}
	}

	measurements := entities.FlattenEnvelope(env)
	alerts := make([]entities.Alert, 0)

	for _, m := range measurements {
		rules, err := e.rulesRepo.FindEnabledByMetric(ctx, m.Metric)
		if err != nil {
			return nil, err
		}

		for _, rule := range rules {
			if !ruleApplies(rule, m) {
				continue
			}

			ok, err := evaluateCondition(rule, m)
			if err != nil {
				return nil, err
			}

			switch rule.Evaluation.Type {
			case entities.EvaluationInstant:
				if ok {
					alerts = append(alerts, buildAlert(rule, m))
				}
			case entities.EvaluationContinuous:
				if triggered, err := e.evaluateContinuous(ctx, rule, m, ok); err != nil {
					return nil, err
				} else if triggered {
					alerts = append(alerts, buildAlert(rule, m))
				}
			case entities.EvaluationInactivity:
				// handled by liveness worker
			default:
				return nil, fmt.Errorf("unknown evaluation type: %s", rule.Evaluation.Type)
			}
		}
	}

	return alerts, nil
}

func (e *RuleEngine) SaveAlerts(ctx context.Context, alerts []entities.Alert) error {
	return e.alertsRepo.InsertMany(ctx, alerts)
}

func (e *RuleEngine) ReleaseMessageMark(ctx context.Context, messageID string) {
	if messageID == "" {
		return
	}
	_ = e.state.DeleteMessageMark(ctx, messageID)
}

func (e *RuleEngine) evaluateContinuous(ctx context.Context, rule entities.Rule, m entities.Measurement, matched bool) (bool, error) {
	if rule.Evaluation.Window == nil {
		return false, errors.New("continuous rule missing window")
	}
	w := rule.Evaluation.Window
	if w.Size <= 0 || w.RequiredMatches <= 0 {
		return false, fmt.Errorf("invalid window config size=%d required=%d", w.Size, w.RequiredMatches)
	}

	switch w.Kind {
	case entities.WindowCount:
		return e.evalCountWindow(ctx, rule, m, matched, w.Size, w.RequiredMatches)
	case entities.WindowTime:
		return e.evalTimeWindow(ctx, rule, m, matched, time.Duration(w.Size)*time.Second, w.RequiredMatches)
	default:
		return false, fmt.Errorf("unknown window kind: %s", w.Kind)
	}
}

func (e *RuleEngine) evalCountWindow(ctx context.Context, rule entities.Rule, m entities.Measurement, matched bool, size, required int) (bool, error) {
	state, _, err := e.state.GetRuleState(ctx, rule.ID, m.Metric, m.DeviceID)
	if err != nil {
		return false, err
	}

	if matched {
		state.ConsecutiveCount++
	} else {
		state.ConsecutiveCount = 0
	}

	val := 0
	if matched {
		val = 1
	}
	state.WindowValues = append(state.WindowValues, val)
	if len(state.WindowValues) > size {
		state.WindowValues = state.WindowValues[len(state.WindowValues)-size:]
	}

	matches := sum(state.WindowValues)
	triggered := len(state.WindowValues) >= size && matches >= required

	state.LastTS = m.Ts
	if triggered {
		state = rstate.RuleState{}
	}

	ttl := time.Hour
	if err := e.state.SetRuleState(ctx, rule.ID, m.Metric, m.DeviceID, state, ttl); err != nil {
		return false, err
	}

	return triggered, nil
}

func (e *RuleEngine) evalTimeWindow(ctx context.Context, rule entities.Rule, m entities.Measurement, matched bool, window time.Duration, required int) (bool, error) {
	if !matched {
		return false, nil
	}

	eventTs := m.Ts
	if eventTs.IsZero() {
		eventTs = time.Now().UTC()
	}

	member := m.DeviceID
	if eventTsUnix := eventTs.UnixNano(); eventTsUnix > 0 {
		member = fmt.Sprintf("%s:%d", m.DeviceID, eventTsUnix)
	}

	if err := e.state.AddToTimeWindow(ctx, rule.ID, m.Metric, m.DeviceID, member, eventTs, window*2); err != nil {
		return false, err
	}

	from := eventTs.Add(-window)
	if err := e.state.PruneTimeWindow(ctx, rule.ID, m.Metric, m.DeviceID, from); err != nil {
		return false, err
	}

	count, err := e.state.CountTimeWindow(ctx, rule.ID, m.Metric, m.DeviceID)
	if err != nil {
		return false, err
	}

	triggered := count >= int64(required)
	if triggered {
		if err := e.state.DeleteTimeWindow(ctx, rule.ID, m.Metric, m.DeviceID); err != nil {
			return false, err
		}
	}
	return triggered, nil
}

func ruleApplies(rule entities.Rule, m entities.Measurement) bool {
	if len(rule.Scope.Metrics) > 0 && !contains(rule.Scope.Metrics, m.Metric) {
		return false
	}
	for k, v := range rule.Scope.Tags {
		if mv, ok := m.Tags[k]; !ok || mv != v {
			return false
		}
	}
	return true
}

func evaluateCondition(rule entities.Rule, m entities.Measurement) (bool, error) {
	if rule.Condition.Metric != "" && rule.Condition.Metric != m.Metric {
		return false, nil
	}

	op := rule.Condition.Op
	val := m.Value
	thr := rule.Condition.Threshold

	switch op {
	case ">":
		return val > thr, nil
	case ">=":
		return val >= thr, nil
	case "<":
		return val < thr, nil
	case "<=":
		return val <= thr, nil
	case "==":
		return val == thr, nil
	case "!=":
		return val != thr, nil
	default:
		return false, fmt.Errorf("unsupported op: %s", op)
	}
}

func buildAlert(rule entities.Rule, m entities.Measurement) entities.Alert {
	return entities.Alert{
		Ts:        m.Ts,
		DeviceID:  m.DeviceID,
		RuleID:    rule.ID,
		Metric:    m.Metric,
		Value:     m.Value,
		Tags:      m.Tags,
		Severity:  rule.Action.Severity,
		Message:   rule.Action.Message,
		MessageID: m.MessageID,
	}
}

func sum(vals []int) int {
	s := 0
	for _, v := range vals {
		s += v
	}
	return s
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
