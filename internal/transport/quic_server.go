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
	connMutex   sync.RWMutex
)

type RegisterMessage struct {
	DeviceID string `json:"device_id"`
	Type     string `json:"type"` // e.g. "register"
}

func StartQUICServer(addr string) error {
	tlsConf := generateTLSConfig()

	listener, err := quic.ListenAddr(addr, tlsConf, nil)
	if err != nil {
		return err
	}

	fmt.Println("QUIC server listening on", addr)

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go handleQUICConnection(conn)
	}
}

func handleQUICConnection(conn quic.Connection) {
	defer conn.CloseWithError(0, "")

	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		log.Println("Initial stream accept error:", err)
		return
	}
	defer stream.Close()

	var msg RegisterMessage
	if err := json.NewDecoder(stream).Decode(&msg); err != nil {
		log.Println("Invalid register message:", err)
		return
	}

	if msg.Type == "register" && msg.DeviceID != "" {
		connMutex.Lock()
		connections[msg.DeviceID] = conn
		connMutex.Unlock()

		fmt.Println("Device registered over QUIC:", msg.DeviceID)
	} else {
		log.Println("Invalid registration message format")
	}
}
// func handleQUICStream(stream quic.Stream) {
// 	defer stream.Close()

// 	var msg map[string]interface{}
// 	decoder := json.NewDecoder(stream)
// 	if err := decoder.Decode(&msg); err != nil {
// 		log.Println("Failed to decode JSON from QUIC stream:", err)
// 		return
// 	}

// 	fmt.Println("Received message over QUIC:", msg)

// 	resp := map[string]string{"status": "ok"}
// 	encoder := json.NewEncoder(stream)
// 	encoder.Encode(resp)
// }

func generateTLSConfig() *tls.Config {
	certFile := "certs/cert.pem"
	keyFile := "certs/key.pem"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("failed to load TLS cert/key: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"pushnova-quic"},
	}
}
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