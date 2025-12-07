package http

import (
	"bytes"
	"data_simulator/entities"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HTTPClient struct {
	baseURL string
	client  http.Client
}

func New(timeout time.Duration, baseURL string) *HTTPClient {
	return &HTTPClient{
		client: http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

func (c *HTTPClient) SendDeviceEvent(payload entities.Event) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, "telemetry")

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return fmt.Errorf("server sent an invalid status code, code: %d", resp.StatusCode)
	}

	return nil
}
