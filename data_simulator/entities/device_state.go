package entities

type DeviceState struct {
	FanSpeed   int     `json:"fan_speed"`
	Mode       string  `json:"mode"`
	FilterLife float64 `json:"filter_life"`
}
