package entities

import (
	"encoding/json"
	"time"
)

type Event struct {
	Timestamp time.Time       `json:"ts"`
	Device    Device          `json:"device"`
	Metrics   Metrics         `json:"metrics"`
	State     DeviceState     `json:"state"`
	Raw       json.RawMessage `json:"raw"`
}
