package entities

import "time"

type Alert struct {
	ID        string            `bson:"_id,omitempty"`
	Ts        time.Time         `bson:"ts"`
	DeviceID  string            `bson:"device_id"`
	RuleID    string            `bson:"rule_id"`
	Metric    string            `bson:"metric"`
	Value     float64           `bson:"value"`
	Tags      map[string]string `bson:"tags"`
	Severity  string            `bson:"severity"`
	Message   string            `bson:"message"`
	Extra     map[string]any    `bson:"extra,omitempty"`
	MessageID string            `bson:"message_id,omitempty"`
}
