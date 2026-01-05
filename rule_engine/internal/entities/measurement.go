package entities

import "time"

type Measurement struct {
	Ts        time.Time
	DeviceID  string
	Metric    string
	Value     float64
	Unit      string
	Tags      map[string]string
	MessageID string
}

func FlattenEnvelope(env Envelope) []Measurement {
	baseTags := map[string]string{
		"device_id":   env.Device.ID,
		"device_type": env.Device.Type,
		"location":    env.Device.Location,
	}

	out := make([]Measurement, 0, len(env.Metrics))
	for name, m := range env.Metrics {
		tagsCopy := copyMap(baseTags)
		out = append(out, Measurement{
			Ts:        env.Ts,
			DeviceID:  env.Device.ID,
			Metric:    name,
			Value:     m.Value,
			Unit:      m.Unit,
			Tags:      tagsCopy,
			MessageID: env.MessageID,
		})
	}
	return out
}

func copyMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
