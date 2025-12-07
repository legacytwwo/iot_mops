package generator

import (
	"data_simulator/entities"
	"math"
	"math/rand"
	"time"
)

func GenerateRandomEvent(deviceID string) entities.Event {
	temp := 18.0 + rand.Float64()*12.0
	hum := 30.0 + rand.Float64()*50.0
	pm25 := 5.0 + rand.Float64()*45.0
	fanSpeed := 1 + rand.Intn(10)
	filterLife := math.Round(rand.Float64()*100) / 100

	return entities.Event{
		Timestamp: time.Now().UTC(),
		Device: entities.Device{
			ID:       deviceID,
			Type:     "dyson_purifier_v2",
			Location: getRandomLocation(),
		},
		Metrics: entities.Metrics{
			Humidity: entities.Metric{
				Value: hum,
				Unit:  "%",
			},
			Temperature: entities.Metric{
				Value: temp,
				Unit:  "C",
			},
			Pm25: entities.Metric{
				Value: pm25,
				Unit:  "µg/m³",
			},
		},
		State: entities.DeviceState{
			FanSpeed:   fanSpeed,
			Mode:       "auto",
			FilterLife: filterLife,
		},
	}
}

func getRandomLocation() string {
	locs := []string{"living-room", "bedroom", "kitchen", "office"}
	return locs[rand.Intn(len(locs))]
}
