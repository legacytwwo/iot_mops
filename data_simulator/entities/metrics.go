package entities

type Metric struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Metrics struct {
	Humidity    Metric `json:"humidity"`
	Temperature Metric `json:"temperature"`
	Pm25        Metric `json:"pm2_5"`
}
