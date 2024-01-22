package redis

import "time"

// msgBusSetKey 集合 key
func msgBusSetKey(topic string) string {
	return "hmsgbus:set:v3:" + topic
}

// 队列key
func msgBusListKey(topic string, channel string) string {
	return "hmsgbus:list:v3:" + topic + ":" + channel
}

const (
	tenMinute = time.Minute * 10
)
