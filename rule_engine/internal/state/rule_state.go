package state

import "time"

type RuleState struct {
	ConsecutiveCount int       `json:"consecutive_count"`
	WindowValues     []int     `json:"window_values,omitempty"`
	LastTS           time.Time `json:"last_ts"`
}
