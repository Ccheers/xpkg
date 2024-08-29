package redis

import (
	"fmt"
	"time"
)

// msgBusSetKey 集合 key
func msgBusSetKey(topic string) string {
	return "hmsgbus:set:v3:" + topic
}

// 队列key
func msgBusListKey(topic string, channel string) string {
	return "hmsgbus:list:v3:" + topic + ":" + channel
}

// Ack key
func msgBusAckKeyPrefix(tm time.Time) string {
	const (
		format = "2006-01-02-15-04"
	)
	return fmt.Sprintf("hmsgbus:hash:v3:ack:%s:", tm.Format(format))
}

// Ack key
func msgBusAckKey(tm time.Time, key string) string {
	return msgBusAckKeyPrefix(tm) + key
}

// Ack key
func msgBusMonitorKey() string {
	return "hmsgbus:nx:v3:monitor"
}

const (
	tenMinute = time.Minute * 10
)
