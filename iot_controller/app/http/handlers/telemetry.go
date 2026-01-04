package handlers

import (
	"encoding/json"
	"iot_controller/entities"
	"net/http"

	"github.com/go-chi/render"
)

func (h *httpHandlers) saveTelemetry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var deviceEvent entities.Event
		if err := json.NewDecoder(r.Body).Decode(&deviceEvent); err != nil {
			render.Status(r, http.StatusBadRequest)
			return
		}

		if err := h.validator.Struct(deviceEvent); err != nil {
			render.Status(r, http.StatusBadRequest)
			return
		}

		if err := h.service.ProcessTelemetry(ctx, &deviceEvent); err != nil {
			render.Status(r, http.StatusInternalServerError)
			return
		}

		render.Status(r, http.StatusOK)
	}
}
