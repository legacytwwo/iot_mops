package entities

import "time"

type Envelope struct {
	MessageID string                 `bson:"message_id,omitempty"`
	Ts        time.Time              `bson:"ts"`
	Device    DeviceInfo             `bson:"device"`
	Metrics   map[string]MetricValue `bson:"metrics"`
	State     map[string]any         `bson:"state,omitempty"`
	Raw       map[string]any         `bson:"raw,omitempty"`
}

type DeviceInfo struct {
	ID       string `bson:"id"`
	Type     string `bson:"type"`
	Location string `bson:"location"`
}

type MetricValue struct {
	Value float64 `bson:"value"`
	Unit  string  `bson:"unit"`
}
