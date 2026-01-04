package entities

type Metric struct {
	Value float64 `json:"value" bson:"value" validate:"required"`
	Unit  string  `json:"unit" bson:"unit" validate:"required,min=1,max=10"`
}

type Metrics struct {
	Humidity    Metric `json:"humidity" bson:"humidity" validate:"required"`
	Temperature Metric `json:"temperature" bson:"temperature" validate:"required"`
	Pm25        Metric `json:"pm2_5" bson:"pm2_5" validate:"required"`
}
