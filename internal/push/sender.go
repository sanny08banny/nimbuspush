package api

import (
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/confluentinc/confluent-kafka-go/kafka"
)

type PublishRequest struct {
    DeviceID string          `json:"device_id"`
    Message  json.RawMessage `json:"message"` // Accept arbitrary JSON object
}

type PublishResponse struct {
    Status string `json:"status"`
}

var Producer *kafka.Producer
var KafkaTopic string

func PublishMessage(w http.ResponseWriter, r *http.Request) {
    var req PublishRequest
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&req); err != nil {
        log.Printf("[ERROR] Invalid JSON from %s: %v", r.RemoteAddr, err)
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    if req.DeviceID == "" || len(req.Message) == 0 {
        log.Printf("[WARN] Missing device_id or message from %s", r.RemoteAddr)
        http.Error(w, "Missing device_id or message", http.StatusBadRequest)
        return
    }

    log.Printf("[INFO] Received message for device: %s from %s", req.DeviceID, r.RemoteAddr)
    log.Printf("[DEBUG] Raw message for %s: %s", req.DeviceID, string(req.Message))

    // Build the full message struct
    kafkaMsg := struct {
        DeviceID  string          `json:"device_id"`
        Message   json.RawMessage `json:"message"` // renamed from "content"
        Timestamp int64           `json:"timestamp"`
    }{
        DeviceID:  req.DeviceID,
        Message:   req.Message,
        Timestamp: time.Now().Unix(),
    }

    jsonBytes, err := json.Marshal(kafkaMsg)
    if err != nil {
        log.Printf("[ERROR] Failed to marshal Kafka message for %s: %v", req.DeviceID, err)
        http.Error(w, "Failed to encode Kafka message", http.StatusInternalServerError)
        return
    }

    msg := &kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &KafkaTopic, Partition: kafka.PartitionAny},
        Key:            []byte(req.DeviceID),
        Value:          jsonBytes,
    }

    err = Producer.Produce(msg, nil)
    if err != nil {
        log.Printf("[ERROR] Failed to send message to Kafka for %s: %v", req.DeviceID, err)
        http.Error(w, "Failed to produce Kafka message", http.StatusInternalServerError)
        return
    }

    log.Printf("[SUCCESS] Message queued for device: %s", req.DeviceID)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(PublishResponse{Status: "Message published"})
}
