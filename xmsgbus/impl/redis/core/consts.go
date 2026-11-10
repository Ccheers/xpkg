package core

import "encoding/base64"

const (
	pendingDataKey = "hmsgbus:v4:pending:data"
	pendingDueKey  = "hmsgbus:v4:pending:due"
)

type channelKeys struct {
	catalog string
	leases  string
	ready   string
}

func catalogKey(topic string) string {
	return "hmsgbus:v4:channels:" + base64.RawURLEncoding.EncodeToString([]byte(topic))
}

func channelKeysFor(topic, channel string) channelKeys {
	prefix := "hmsgbus:v4:channel:" + base64.RawURLEncoding.EncodeToString([]byte(topic)) +
		":" + base64.RawURLEncoding.EncodeToString([]byte(channel))
	return channelKeys{catalog: catalogKey(topic), leases: prefix + ":leases", ready: prefix + ":ready"}
}
