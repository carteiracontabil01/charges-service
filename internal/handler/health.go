package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

// Health godoc
// @Summary      Health check
// @Description  Check the health of the service
// @Tags         status
// @Success      200  {object}  HealthResponse
// @Router       /health [get]
func Health(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{Status: "ok", Time: time.Now()}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
