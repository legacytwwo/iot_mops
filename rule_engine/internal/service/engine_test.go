package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"rule_engine/internal/entities"
	rstate "rule_engine/internal/state"
)

type fakeRulesRepo struct {
	rules []entities.Rule
}

func (f fakeRulesRepo) FindEnabledByMetric(_ context.Context, metric string) ([]entities.Rule, error) {
	out := make([]entities.Rule, 0)
	for _, r := range f.rules {
		if !r.Enabled {
			continue
		}
		if len(r.Scope.Metrics) == 0 {
			out = append(out, r)
			continue
		}
		for _, m := range r.Scope.Metrics {
			if m == metric {
				out = append(out, r)
				break
			}
		}
	}
	return out, nil
}

func (f fakeRulesRepo) FindInactivityRules(_ context.Context) ([]entities.Rule, error) {
	return nil, nil
}

type fakeAlertsRepo struct {
	mu     sync.Mutex
	alerts []entities.Alert
}

func (f *fakeAlertsRepo) Insert(_ context.Context, alert entities.Alert) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alerts = append(f.alerts, alert)
	return nil
}

func (f *fakeAlertsRepo) InsertMany(_ context.Context, alerts []entities.Alert) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alerts = append(f.alerts, alerts...)
	return nil
}

type memoryState struct {
	mu       sync.Mutex
	states   map[string]rstate.RuleState
	windows  map[string][]time.Time
	messages map[string]struct{}
}

func newMemoryState() *memoryState {
	return &memoryState{
		states:   make(map[string]rstate.RuleState),
		windows:  make(map[string][]time.Time),
		messages: make(map[string]struct{}),
	}
}

func (m *memoryState) GetRuleState(_ context.Context, ruleID, metric, deviceID string) (rstate.RuleState, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ruleKey(ruleID, metric, deviceID)
	state, ok := m.states[key]
	return state, ok, nil
}

func (m *memoryState) SetRuleState(_ context.Context, ruleID, metric, deviceID string, state rstate.RuleState, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ruleKey(ruleID, metric, deviceID)
	m.states[key] = state
	return nil
}

func (m *memoryState) DeleteRuleState(_ context.Context, ruleID, metric, deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ruleKey(ruleID, metric, deviceID)
	delete(m.states, key)
	return nil
}

func (m *memoryState) AddToTimeWindow(_ context.Context, ruleID, metric, deviceID, _ string, ts time.Time, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := windowKey(ruleID, metric, deviceID)
	m.windows[key] = append(m.windows[key], ts)
	return nil
}

func (m *memoryState) PruneTimeWindow(_ context.Context, ruleID, metric, deviceID string, from time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := windowKey(ruleID, metric, deviceID)
	vals := m.windows[key]
	kept := vals[:0]
	for _, ts := range vals {
		if !ts.Before(from) {
			kept = append(kept, ts)
		}
	}
	if len(kept) == 0 {
		delete(m.windows, key)
		return nil
	}
	m.windows[key] = kept
	return nil
}

func (m *memoryState) CountTimeWindow(_ context.Context, ruleID, metric, deviceID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := windowKey(ruleID, metric, deviceID)
	return int64(len(m.windows[key])), nil
}

func (m *memoryState) DeleteTimeWindow(_ context.Context, ruleID, metric, deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := windowKey(ruleID, metric, deviceID)
	delete(m.windows, key)
	return nil
}

func (m *memoryState) GetLastSeen(_ context.Context, _ string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

func (m *memoryState) TryMarkMessage(_ context.Context, messageID string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.messages[messageID]; ok {
		return false, nil
	}
	m.messages[messageID] = struct{}{}
	return true, nil
}

func (m *memoryState) DeleteMessageMark(_ context.Context, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.messages, messageID)
	return nil
}

func ruleKey(ruleID, metric, deviceID string) string {
	return ruleID + "|" + metric + "|" + deviceID
}

func windowKey(ruleID, metric, deviceID string) string {
	return ruleID + "|" + metric + "|" + deviceID + "|window"
}

func TestRuleEngineInstantRule(t *testing.T) {
	rule := entities.Rule{
		ID:      "rule_instant",
		Enabled: true,
		Scope:   entities.RuleScope{Metrics: []string{"temperature"}},
		Condition: entities.Condition{
			Metric:    "temperature",
			Op:        ">",
			Threshold: 30,
		},
		Evaluation: entities.Evaluation{Type: entities.EvaluationInstant},
		Action:     entities.RuleAction{Severity: "warning", Message: "hot"},
	}

	env := entities.Envelope{
		Ts: time.Now().UTC(),
		Device: entities.DeviceInfo{
			ID:       "dev-1",
			Type:     "sensor",
			Location: "room",
		},
		Metrics: map[string]entities.MetricValue{
			"temperature": {Value: 35, Unit: "C"},
		},
	}

	engine := NewRuleEngine(fakeRulesRepo{rules: []entities.Rule{rule}}, &fakeAlertsRepo{}, newMemoryState(), zap.NewNop(), time.Hour)
	alerts, err := engine.ProcessEnvelope(context.Background(), env)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].RuleID != rule.ID {
		t.Fatalf("unexpected rule id: %s", alerts[0].RuleID)
	}
}

func TestRuleEngineCountWindow(t *testing.T) {
	rule := entities.Rule{
		ID:      "rule_count",
		Enabled: true,
		Scope:   entities.RuleScope{Metrics: []string{"humidity"}},
		Condition: entities.Condition{
			Metric:    "humidity",
			Op:        ">",
			Threshold: 50,
		},
		Evaluation: entities.Evaluation{
			Type: entities.EvaluationContinuous,
			Window: &entities.Window{
				Kind:            entities.WindowCount,
				Size:            3,
				RequiredMatches: 2,
			},
		},
		Action: entities.RuleAction{Severity: "warning", Message: "humid"},
	}

	engine := NewRuleEngine(fakeRulesRepo{rules: []entities.Rule{rule}}, &fakeAlertsRepo{}, newMemoryState(), zap.NewNop(), time.Hour)
	ctx := context.Background()

	values := []float64{60, 40, 70}
	alertsTotal := 0
	for _, v := range values {
		env := entities.Envelope{
			Ts:     time.Now().UTC(),
			Device: entities.DeviceInfo{ID: "dev-1", Type: "sensor", Location: "room"},
			Metrics: map[string]entities.MetricValue{
				"humidity": {Value: v, Unit: "%"},
			},
		}
		alerts, err := engine.ProcessEnvelope(ctx, env)
		if err != nil {
			t.Fatalf("process: %v", err)
		}
		alertsTotal += len(alerts)
	}

	if alertsTotal != 1 {
		t.Fatalf("expected 1 alert after window, got %d", alertsTotal)
	}
}

func TestRuleEngineTimeWindow(t *testing.T) {
	rule := entities.Rule{
		ID:      "rule_time",
		Enabled: true,
		Scope:   entities.RuleScope{Metrics: []string{"pm2_5"}},
		Condition: entities.Condition{
			Metric:    "pm2_5",
			Op:        ">",
			Threshold: 10,
		},
		Evaluation: entities.Evaluation{
			Type: entities.EvaluationContinuous,
			Window: &entities.Window{
				Kind:            entities.WindowTime,
				Size:            10,
				RequiredMatches: 2,
			},
		},
		Action: entities.RuleAction{Severity: "warning", Message: "air"},
	}

	engine := NewRuleEngine(fakeRulesRepo{rules: []entities.Rule{rule}}, &fakeAlertsRepo{}, newMemoryState(), zap.NewNop(), time.Hour)
	ctx := context.Background()

	base := time.Now().UTC()
	for i, ts := range []time.Time{base, base.Add(5 * time.Second)} {
		env := entities.Envelope{
			Ts:     ts,
			Device: entities.DeviceInfo{ID: "dev-1", Type: "sensor", Location: "room"},
			Metrics: map[string]entities.MetricValue{
				"pm2_5": {Value: 20, Unit: "ug/m3"},
			},
		}
		alerts, err := engine.ProcessEnvelope(ctx, env)
		if err != nil {
			t.Fatalf("process: %v", err)
		}
		if i == 0 && len(alerts) != 0 {
			t.Fatalf("expected 0 alerts on first event")
		}
		if i == 1 && len(alerts) != 1 {
			t.Fatalf("expected 1 alert on second event, got %d", len(alerts))
		}
	}
}

func TestRuleEngineIdempotency(t *testing.T) {
	rule := entities.Rule{
		ID:      "rule_idem",
		Enabled: true,
		Scope:   entities.RuleScope{Metrics: []string{"temperature"}},
		Condition: entities.Condition{
			Metric:    "temperature",
			Op:        ">",
			Threshold: 30,
		},
		Evaluation: entities.Evaluation{Type: entities.EvaluationInstant},
		Action:     entities.RuleAction{Severity: "warning", Message: "hot"},
	}

	engine := NewRuleEngine(fakeRulesRepo{rules: []entities.Rule{rule}}, &fakeAlertsRepo{}, newMemoryState(), zap.NewNop(), time.Hour)
	ctx := context.Background()

	env := entities.Envelope{
		MessageID: "msg-1",
		Ts:        time.Now().UTC(),
		Device:    entities.DeviceInfo{ID: "dev-1", Type: "sensor", Location: "room"},
		Metrics: map[string]entities.MetricValue{
			"temperature": {Value: 35, Unit: "C"},
		},
	}

	alerts, err := engine.ProcessEnvelope(ctx, env)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	alerts, err = engine.ProcessEnvelope(ctx, env)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts for duplicate message, got %d", len(alerts))
	}
}
