package dto

import (
	"fmt"
	"time"

	"rule_engine/internal/entities"
)

type EnvelopeDTO struct {
	MessageID string               `json:"message_id,omitempty"`
	Ts        string               `json:"ts"`
	Device    DeviceInfoDTO        `json:"device"`
	Metrics   map[string]MetricDTO `json:"metrics"`
	State     map[string]any       `json:"state,omitempty"`
	Raw       map[string]any       `json:"raw,omitempty"`
}

type DeviceInfoDTO struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Location string `json:"location"`
}

type MetricDTO struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

func (d EnvelopeDTO) ToEntity() (entities.Envelope, error) {
	ts, err := time.Parse(time.RFC3339, d.Ts)
	if err != nil {
		return entities.Envelope{}, fmt.Errorf("parse ts: %w", err)
	}

	metrics := make(map[string]entities.MetricValue, len(d.Metrics))
	for name, m := range d.Metrics {
		metrics[name] = entities.MetricValue{
			Value: m.Value,
			Unit:  m.Unit,
		}
	}

	return entities.Envelope{
		MessageID: d.MessageID,
		Ts:        ts,
		Device: entities.DeviceInfo{
			ID:       d.Device.ID,
			Type:     d.Device.Type,
			Location: d.Device.Location,
		},
		Metrics: metrics,
		State:   d.State,
		Raw:     d.Raw,
	}, nil
}
