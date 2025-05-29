package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"nimbuspush/config"

	"gorm.io/gorm"
)

type Device struct {
	gorm.Model
	DeviceID string `gorm:"uniqueIndex;not null" json:"device_id"`
	Token    string `gorm:"not null" json:"token"`
}

type RegisterRequest struct {
	DeviceID string `json:"device_id"`
	Token    string `json:"token"`
}

type RegisterResponse struct {
	Message string `json:"message"`
}

func RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" || req.Token == "" {
		http.Error(w, "Missing device_id or token", http.StatusBadRequest)
		return
	}

	device := Device{
		DeviceID: req.DeviceID,
		Token:    req.Token,
	}

	if err := upsertDevice(&device); err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RegisterResponse{
		Message: "Device registered or updated",
	})
}

func upsertDevice(device *Device) error {
	db := config.DB

	var existing Device
	err := db.Where("device_id = ?", device.DeviceID).First(&existing).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Println("DB lookup failed:", err)
		return err
	}

	if existing.ID != 0 {
		log.Println("Device exists, updating token")
		existing.Token = device.Token
		return db.Save(&existing).Error
	}

	log.Println("Device not found, creating new one")
	return db.Create(device).Error
}



