package entities

import (
	"encoding/json"
	"time"
)

type Event struct {
	Timestamp time.Time       `json:"ts" bson:"ts" validate:"required"`
	Device    Device          `json:"device" bson:"device" validate:"required"`
	Metrics   Metrics         `json:"metrics" bson:"metrics" validate:"required"`
	State     DeviceState     `json:"state" bson:"state" validate:"required"`
	Raw       json.RawMessage `json:"raw" bson:"raw,omitempty" validate:"omitempty,json"`
}
