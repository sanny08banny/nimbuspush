package api

import (
    "encoding/json"
    "net/http"

    "github.com/confluentinc/confluent-kafka-go/kafka"
)

type PublishRequest struct {
    DeviceID string `json:"device_id"`
    Message  string `json:"message"`
}

type PublishResponse struct {
    Status string `json:"status"`
}

var Producer *kafka.Producer
var KafkaTopic string

func PublishMessage(w http.ResponseWriter, r *http.Request) {
    var req PublishRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    if req.DeviceID == "" || req.Message == "" {
        http.Error(w, "Missing device_id or message", http.StatusBadRequest)
        return
    }

    msg := &kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &KafkaTopic, Partition: kafka.PartitionAny},
        Key:            []byte(req.DeviceID),
        Value:          []byte(req.Message),
    }

    err := Producer.Produce(msg, nil)
    if err != nil {
        http.Error(w, "Failed to produce Kafka message", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(PublishResponse{Status: "Message published"})
}
