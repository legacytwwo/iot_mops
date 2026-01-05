package liveness

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"rule_engine/internal/entities"
	"rule_engine/internal/metrics"
)

type RulesRepository interface {
	FindInactivityRules(ctx context.Context) ([]entities.Rule, error)
}

type DevicesRepository interface {
	FindIDsByScope(ctx context.Context, scope entities.DeviceScope) ([]string, error)
}

type StateStore interface {
	GetLastSeen(ctx context.Context, deviceID string) (time.Time, bool, error)
}

type AlertsRepository interface {
	Insert(ctx context.Context, alert entities.Alert) error
	InsertMany(ctx context.Context, alerts []entities.Alert) error
}

type Worker struct {
	interval time.Duration
	rules    RulesRepository
	devices  DevicesRepository
	state    StateStore
	alerts   AlertsRepository
	metrics  *metrics.PromMetrics
	logger   *zap.Logger
}

func New(interval time.Duration, rules RulesRepository, devices DevicesRepository, state StateStore, alerts AlertsRepository, metrics *metrics.PromMetrics, logger *zap.Logger) *Worker {
	return &Worker{
		interval: interval,
		rules:    rules,
		devices:  devices,
		state:    state,
		alerts:   alerts,
		metrics:  metrics,
		logger:   logger,
	}
}

func (w *Worker) Start(ctx context.Context) {
	if w.interval <= 0 {
		w.interval = 5 * time.Second
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	w.metrics.IncLivenessCheck()

	rules, err := w.rules.FindInactivityRules(ctx)
	if err != nil {
		w.metrics.IncLivenessError("rules")
		w.logger.Error("liveness rules", zap.Error(err))
		return
	}

	alerts := make([]entities.Alert, 0)
	for _, rule := range rules {
		if rule.Evaluation.Inactivity == nil {
			continue
		}
		gap := time.Duration(rule.Evaluation.Inactivity.MaxGapSec) * time.Second
		if gap <= 0 {
			continue
		}

		scope := entities.DeviceScope{
			IDs:      rule.Scope.DeviceIDs,
			Type:     rule.Scope.Tags["device_type"],
			Location: rule.Scope.Tags["location"],
		}
		deviceIDs, err := w.devices.FindIDsByScope(ctx, scope)
		if err != nil {
			w.metrics.IncLivenessError("devices")
			w.logger.Error("liveness devices", zap.Error(err))
			continue
		}

		for _, deviceID := range deviceIDs {
			lastSeen, ok, err := w.state.GetLastSeen(ctx, deviceID)
			if err != nil {
				w.metrics.IncLivenessError("last_seen")
				w.logger.Error("liveness last seen", zap.String("device_id", deviceID), zap.Error(err))
				continue
			}
			if !ok {
				continue
			}
			if time.Since(lastSeen) < gap {
				continue
			}

			alerts = append(alerts, entities.Alert{
				Ts:       time.Now().UTC(),
				DeviceID: deviceID,
				RuleID:   rule.ID,
				Metric:   "inactivity",
				Value:    0,
				Tags:     rule.Scope.Tags,
				Severity: rule.Action.Severity,
				Message:  rule.Action.Message,
			})
		}
	}

	if len(alerts) == 0 {
		return
	}

	if err := w.alerts.InsertMany(ctx, alerts); err != nil {
		w.metrics.IncLivenessError("alerts_save")
		w.logger.Error("liveness save alerts", zap.Error(err))
		return
	}

	w.metrics.IncLivenessAlert(len(alerts))
}

var ErrNoRules = errors.New("no inactivity rules")
