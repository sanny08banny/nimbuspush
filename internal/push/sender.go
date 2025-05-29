package api

import (
	"encoding/json"
	"net/http"
	"nimbuspush/internal/transport"
)

type PushRequest struct {
	DeviceID string `json:"device_id"`
	Title    string `json:"title"`
	Message  string `json:"message"`
}

type PushResponse struct {
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

func PushMessage(w http.ResponseWriter, r *http.Request) {
	var req PushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" || req.Title == "" || req.Message == "" {
		http.Error(w, "Missing fields: device_id, title, or message", http.StatusBadRequest)
		return
	}

	// payload := map[string]string{
	// 	"title":   req.Title,
	// 	"message": req.Message,
	// }

	err := transport.SendPush(req.DeviceID, req.Message)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(PushResponse{
			Status:  "error",
			Details: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(PushResponse{
		Status: "success",
	})
}
