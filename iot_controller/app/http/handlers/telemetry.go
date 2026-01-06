package handlers

import (
	"encoding/json"
	"iot_controller/entities"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/render"
)

func (h *httpHandlers) saveTelemetry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.metrics.IncMessage("received")
		ctx := r.Context()

		var deviceEvent entities.Event
		if err := json.NewDecoder(r.Body).Decode(&deviceEvent); err != nil {
			h.metrics.IncValidationErrors()
			log.Println("failed to decode request body: %w", err)
			render.Status(r, http.StatusBadRequest)
			return
		}

		if err := h.validator.Struct(deviceEvent); err != nil {
			h.metrics.IncValidationErrors()
			log.Println("failed to validate device event: %w", err)
			render.Status(r, http.StatusBadRequest)
			return
		}

		if err := h.service.ProcessTelemetry(ctx, &deviceEvent); err != nil {
			log.Println("failed to process event: %w", err)
			render.Status(r, http.StatusInternalServerError)
			return
		}

		h.metrics.ObserveProcessDuration("total", time.Since(start))

		render.Status(r, http.StatusOK)
	}
}
