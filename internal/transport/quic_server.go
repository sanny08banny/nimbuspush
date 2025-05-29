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

// func handleQUICConnection(conn quic.Connection) {
// 	defer conn.CloseWithError(0, "")

// 	stream, err := conn.AcceptStream(context.Background())
// 	if err != nil {
// 		log.Println("Initial stream accept error:", err)
// 		return
// 	}
// 	defer stream.Close()

// 	var msg RegisterMessage
// 	if err := json.NewDecoder(stream).Decode(&msg); err != nil {
// 		log.Println("Invalid register message:", err)
// 		return
// 	}

// 	if msg.Type == "register" && msg.DeviceID != "" {
// 		connMutex.Lock()
// 		connections[msg.DeviceID] = conn
// 		connMutex.Unlock()

// 		fmt.Println("Device registered over QUIC:", msg.DeviceID)
// 	} else {
// 		log.Println("Invalid registration message format")
// 	}
// }
type DeviceConnection struct {
	DeviceID string
	Stream   quic.Stream
}
func handleQUICConnection(conn quic.Connection) {
    defer func() {
        conn.CloseWithError(0, "connection closed")
        connMutex.Lock()
        // Remove device mapping on disconnect
        for id, c := range connections {
            if c == conn {
                delete(connections, id)
                break
            }
        }
        connMutex.Unlock()
    }()

    for {
        stream, err := conn.AcceptStream(context.Background())
        if err != nil {
            log.Println("Stream accept error:", err)
            return
        }

        go func(s quic.Stream) {
            defer s.Close()
            // handle streams from client if needed (not required now)
        }(stream)
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
var (
	deviceMap = make(map[string]DeviceConnection)
	mutex     = &sync.Mutex{}
)

func StartServer(addr string) error {
	tlsConf := generateTLSConfig()
	

	listener, err := quic.ListenAddr(addr, tlsConf, nil)
	if err != nil {
		return fmt.Errorf("listen addr: %w", err)
	}

	log.Printf("QUIC push server started on %s\n", addr)

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("Accept error: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn quic.Connection) {
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		log.Println("Stream accept error:", err)
		return
	}

	// Read JSON registration
	var msg map[string]string
	if err := json.NewDecoder(stream).Decode(&msg); err != nil {
		log.Println("Failed to decode registration:", err)
		return
	}

	deviceID := msg["device_id"]
	if deviceID == "" || msg["type"] != "register" {
		log.Println("Invalid registration message")
		return
	}

	log.Printf("✅ Device connected: %s\n", deviceID)

	// Save device stream for pushing
	mutex.Lock()
	deviceMap[deviceID] = DeviceConnection{
		DeviceID: deviceID,
		Stream:   stream,
	}
	mutex.Unlock()

	// Optional: send welcome message
	welcome := map[string]string{
		"type":    "welcome",
		"message": "Registered successfully",
	}
	if err := json.NewEncoder(stream).Encode(welcome); err != nil {
		log.Println("Failed to send welcome message:", err)
		return
	}

	// Block forever — keeps stream alive for future pushes
	select {}
}

func SendPush(deviceID, message string) error {
	mutex.Lock()
	defer mutex.Unlock()

	dev, exists := deviceMap[deviceID]
	if !exists {
		return fmt.Errorf("device %s not connected", deviceID)
	}

	_, err := dev.Stream.Write([]byte(message))
	return err
}

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