package xmsgbus

import "google.golang.org/grpc/metadata"

type Event struct {
	Metadata metadata.MD
	Topic    string
	Payload  []byte
}
