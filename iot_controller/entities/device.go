package entities

type Device struct {
	ID       string `json:"id" bson:"id" validate:"required"`
	Type     string `json:"type" bson:"type" validate:"required"`
	Location string `json:"location" bson:"location" validate:"required,oneof=living-room bedroom kitchen office"`
}
