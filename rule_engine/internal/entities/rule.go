package entities

import "time"

type Rule struct {
	ID         string     `bson:"_id"`
	Name       string     `bson:"name"`
	Enabled    bool       `bson:"enabled"`
	Scope      RuleScope  `bson:"scope"`
	Condition  Condition  `bson:"condition,omitempty"`
	Evaluation Evaluation `bson:"evaluation"`
	Action     RuleAction `bson:"action"`
	UpdatedAt  time.Time  `bson:"updated_at"`
}

type RuleScope struct {
	Metrics   []string          `bson:"metrics"`
	Tags      map[string]string `bson:"tags"`
	DeviceIDs []string          `bson:"device_ids,omitempty"`
}

type Condition struct {
	Metric    string  `bson:"metric"`
	Op        string  `bson:"op"`
	Threshold float64 `bson:"threshold"`
}

type Evaluation struct {
	Type       string          `bson:"type"`
	Window     *Window         `bson:"window,omitempty"`
	Inactivity *InactivityRule `bson:"inactivity,omitempty"`
}

type Window struct {
	Kind            string `bson:"kind"`
	Size            int    `bson:"size"`
	RequiredMatches int    `bson:"required_matches"`
}

type InactivityRule struct {
	MaxGapSec int `bson:"max_gap_sec"`
}

type RuleAction struct {
	Type     string `bson:"type"`
	Severity string `bson:"severity"`
	Message  string `bson:"message"`
}

const (
	EvaluationInstant    = "instant"
	EvaluationContinuous = "continuous"
	EvaluationInactivity = "inactivity"

	WindowCount = "count"
	WindowTime  = "time"
)
