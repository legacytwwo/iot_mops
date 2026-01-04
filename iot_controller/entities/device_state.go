package entities

type DeviceState struct {
	FanSpeed   int     `json:"fan_speed" bson:"fan_speed" validate:"min=0,max=100"`
	Mode       string  `json:"mode" bson:"mode" validate:"required,oneof=auto manual sleep turbo off"`
	FilterLife float64 `json:"filter_life" bson:"filter_life" validate:"min=0.0,max=100.0"`
}
