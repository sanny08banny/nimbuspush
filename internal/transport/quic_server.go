package transport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/quic-go/quic-go"
)

var (
	connections = make(map[string]quic.Connection)
	connMutex   = &sync.RWMutex{}
)

type RegisterMessage struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"` // e.g. "register"
	// Add token or auth field here if needed later
}

// StartServer starts the QUIC server and listens for incoming device connections.
func StartServer(addr string) error {
	tlsConf := generateTLSConfig()

	listener, err := quic.ListenAddr(addr, tlsConf, nil)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	log.Printf("✅ QUIC server started on %s", addr)

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("❌ Accept error: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

// handleConnection handles a new QUIC device connection and registers it.
func handleConnection(conn quic.Connection) {
	defer func() {
		conn.CloseWithError(0, "closing connection")
		removeConnection(conn)
	}()

	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		log.Println("❌ Stream accept error:", err)
		return
	}
	defer stream.Close()

	var msg RegisterMessage
	if err := json.NewDecoder(stream).Decode(&msg); err != nil {
		log.Println("❌ Failed to decode registration:", err)
		return
	}

	if msg.Type != "register" || msg.DeviceID == "" {
		log.Println("❌ Invalid registration message")
		return
	}

	connMutex.Lock()
	connections[msg.DeviceID] = conn
	connMutex.Unlock()

	log.Printf("✅ Device registered: %s", msg.DeviceID)

	// Optional welcome message
	welcome := map[string]string{
		"type":    "welcome",
		"message": "Registered successfully",
	}
	if err := json.NewEncoder(stream).Encode(welcome); err != nil {
		log.Println("❌ Failed to send welcome message:", err)
	}

	// Keep connection alive (no-op select)
	select {}
}

// removeConnection removes a connection from the registry.
func removeConnection(conn quic.Connection) {
	connMutex.Lock()
	defer connMutex.Unlock()

	for id, c := range connections {
		if c == conn {
			delete(connections, id)
			log.Printf("🔌 Device disconnected: %s", id)
			break
		}
	}
}

// SendToDevice sends a JSON payload to the registered device.
func SendToDevice(deviceID string, payload interface{}) error {
	connMutex.RLock()
	conn, ok := connections[deviceID]
	connMutex.RUnlock()

	if !ok {
		return fmt.Errorf("device %s not connected", deviceID)
	}

	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	encoder := json.NewEncoder(stream)
	return encoder.Encode(payload)
}

// generateTLSConfig loads TLS certificates and returns a QUIC-compatible TLS config.
func generateTLSConfig() *tls.Config {
	certFile := "certs/cert.pem"
	keyFile := "certs/key.pem"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("❌ Failed to load TLS cert/key: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"pushnova-quic"},
	}
}
