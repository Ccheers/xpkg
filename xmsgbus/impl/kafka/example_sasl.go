package kafka

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
)

// ExampleUsage demonstrates how to use SASL authentication with Kafka
func ExampleUsage() {
	// Example 1: SASL/PLAIN with TLS
	msgBus1, err := NewMsgBus(
		WithBrokers([]string{"kafka1:9093", "kafka2:9093"}),
		WithSASLPlainAuth("username", "password"),
		WithTLS(nil), // Use default TLS config
	)
	if err != nil {
		log.Fatal("Failed to create msgbus with SASL/PLAIN:", err)
	}
	defer msgBus1.(*MsgBus).Close()

	// Example 2: SASL/SCRAM-SHA256 with custom TLS
	tlsConfig := &tls.Config{
		ServerName:         "kafka.example.com",
		InsecureSkipVerify: false,
	}

	msgBus2, err := NewMsgBus(
		WithBrokers([]string{"kafka.example.com:9093"}),
		WithSASLSCRAMSHA256Auth("username", "password"),
		WithTLS(tlsConfig),
	)
	if err != nil {
		log.Fatal("Failed to create msgbus with SASL/SCRAM-SHA256:", err)
	}
	defer msgBus2.(*MsgBus).Close()

	// Example 3: SASL/SCRAM-SHA512 with insecure TLS (for testing)
	msgBus3, err := NewMsgBus(
		WithBrokers([]string{"localhost:9093"}),
		WithSASLSCRAMSHA512Auth("testuser", "testpass"),
		WithInsecureTLS(),
	)
	if err != nil {
		log.Fatal("Failed to create msgbus with SASL/SCRAM-SHA512:", err)
	}
	defer msgBus3.(*MsgBus).Close()

	// Example 4: Storage with SASL authentication
	storage, err := NewStorage(
		[]string{"kafka1:9093", "kafka2:9093"},
		WithSASLPlainAuth("username", "password"),
		WithTLS(nil),
	)
	if err != nil {
		log.Fatal("Failed to create storage with SASL:", err)
	}
	defer storage.(*Storage).Close()

	// Use the msgbus normally
	ctx := context.Background()
	topic := "example_topic"
	channel := "example_channel"

	// Add channel
	err = msgBus1.AddChannel(ctx, topic, channel)
	if err != nil {
		log.Fatal("Failed to add channel:", err)
	}

	// Push message
	err = msgBus1.Push(ctx, topic, []byte("Hello SASL World!"))
	if err != nil {
		log.Fatal("Failed to push message:", err)
	}

	// Pop message
	data, ackFn, err := msgBus1.Pop(ctx, topic, channel, 5*time.Second)
	if err != nil {
		log.Fatal("Failed to pop message:", err)
	}

	log.Printf("Received message: %s", string(data))
	if ackFn != nil {
		ackFn()
	}

	// Use storage
	err = storage.SetEx(ctx, "test_key", "test_value", 30*time.Second)
	if err != nil {
		log.Fatal("Failed to set key:", err)
	}

	keys, err := storage.Keys(ctx, "test_")
	if err != nil {
		log.Fatal("Failed to get keys:", err)
	}

	log.Printf("Found keys: %v", keys)
}

// Production configuration example
func ProductionConfig() xmsgbus.IMsgBus {
	// Production-ready configuration with SASL and TLS
	msgBus, err := NewMsgBus(
		WithBrokers([]string{
			"kafka1.prod.example.com:9093",
			"kafka2.prod.example.com:9093",
			"kafka3.prod.example.com:9093",
		}),
		WithSASLSCRAMSHA256Auth("prod_user", "secure_password"),
		WithTLS(&tls.Config{
			ServerName:         "kafka.prod.example.com",
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}),
		WithGroupPrefix("myapp"),
		WithSessionTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatal("Failed to create production msgbus:", err)
	}

	return msgBus
}
