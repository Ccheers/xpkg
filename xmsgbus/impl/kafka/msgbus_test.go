package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/ccheers/xpkg/xmsgbus"
)

var kafkaEndpoints = []string{}

func TestKafkaMsgBus_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kafka integration test in short mode")
		return
	}
	msgBus, err := NewMsgBus(WithBrokers(kafkaEndpoints))
	if err != nil {
		t.Fatalf("failed to create msgbus: %v", err)
	}
	defer msgBus.(*MsgBus).Close()

	ctx := context.Background()
	topic := "test_topic_1"
	channel := "test_channel"
	testData := []byte("test message")

	err = msgBus.AddChannel(ctx, topic, channel)
	if err != nil {
		t.Fatalf("failed to add channel: %v", err)
	}

	time.Sleep(2 * time.Second)

	err = msgBus.Push(ctx, topic, testData)
	if err != nil {
		t.Fatalf("failed to push message: %v", err)
	}

	data, ackFn, err := msgBus.Pop(ctx, topic, channel, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to pop message: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("expected %s, got %s", testData, data)
	}

	if ackFn != nil {
		ackFn()
	}

	channels, err := msgBus.ListChannel(ctx, topic)
	if err != nil {
		t.Fatalf("failed to list channels: %v", err)
	}

	if len(channels) != 1 || channels[0] != channel {
		t.Errorf("expected [%s], got %v", channel, channels)
	}

	err = msgBus.RemoveChannel(ctx, topic, channel)
	if err != nil {
		t.Fatalf("failed to remove channel: %v", err)
	}

	channels, err = msgBus.ListChannel(ctx, topic)
	if err != nil {
		t.Fatalf("failed to list channels after removal: %v", err)
	}

	if len(channels) != 0 {
		t.Errorf("expected empty channels, got %v", channels)
	}
}

func TestKafkaMsgBus_MultipleChannels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kafka integration test in short mode")
		return
	}

	msgBus, err := NewMsgBus(WithBrokers(kafkaEndpoints))
	if err != nil {
		t.Fatalf("failed to create msgbus: %v", err)
	}
	defer msgBus.(*MsgBus).Close()

	ctx := context.Background()
	topic := "test_multi_topic_1"
	channel1 := "channel1"
	channel2 := "channel2"
	testData := []byte("multi channel test")

	err = msgBus.AddChannel(ctx, topic, channel1)
	if err != nil {
		t.Fatalf("failed to add channel1: %v", err)
	}

	err = msgBus.AddChannel(ctx, topic, channel2)
	if err != nil {
		t.Fatalf("failed to add channel2: %v", err)
	}

	time.Sleep(2 * time.Second)

	err = msgBus.Push(ctx, topic, testData)
	if err != nil {
		t.Fatalf("failed to push message: %v", err)
	}

	data1, ackFn1, err := msgBus.Pop(ctx, topic, channel1, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to pop from channel1: %v", err)
	}

	data2, ackFn2, err := msgBus.Pop(ctx, topic, channel2, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to pop from channel2: %v", err)
	}

	if string(data1) != string(testData) {
		t.Errorf("channel1: expected %s, got %s", testData, data1)
	}

	if string(data2) != string(testData) {
		t.Errorf("channel2: expected %s, got %s", testData, data2)
	}

	if ackFn1 != nil {
		ackFn1()
	}
	if ackFn2 != nil {
		ackFn2()
	}
}

func TestKafkaMsgBus_PopTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kafka integration test in short mode")
		return
	}

	msgBus, err := NewMsgBus(WithBrokers(kafkaEndpoints))
	if err != nil {
		t.Fatalf("failed to create msgbus: %v", err)
	}
	defer msgBus.(*MsgBus).Close()

	ctx := context.Background()
	topic := "test_timeout_topic_1"
	channel := "timeout_channel"

	err = msgBus.AddChannel(ctx, topic, channel)
	if err != nil {
		t.Fatalf("failed to add channel: %v", err)
	}

	time.Sleep(2 * time.Second)

	_, _, err = msgBus.Pop(ctx, topic, channel, 1*time.Second)
	if err != xmsgbus.ErrPopTimeout {
		t.Errorf("expected timeout error, got %v", err)
	}
}
