package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/ccheers/xpkg/xmsgbus"
)

type Storage struct {
	producer sarama.SyncProducer
	admin    sarama.ClusterAdmin
	brokers  []string
	config   *sarama.Config
}

func NewStorage(brokers []string, options ...IMsgBusOption) (xmsgbus.ISharedStorage, error) {
	opts := defaultMsgBusOptions()
	opts.brokers = brokers
	for _, opt := range options {
		opt.apply(&opts)
	}

	producer, err := sarama.NewSyncProducer(opts.brokers, opts.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	admin, err := sarama.NewClusterAdmin(opts.brokers, opts.config)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}

	return &Storage{
		producer: producer,
		admin:    admin,
		brokers:  opts.brokers,
		config:   opts.config,
	}, nil
}

func (s *Storage) SetEx(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	topicName := "xmsgbus_storage"

	if err := s.ensureTopicExists(topicName); err != nil {
		return fmt.Errorf("failed to ensure topic exists: %w", err)
	}

	var valueBytes []byte
	switch v := value.(type) {
	case []byte:
		valueBytes = v
	case string:
		valueBytes = []byte(v)
	default:
		valueBytes = []byte(fmt.Sprintf("%v", v))
	}

	headers := []sarama.RecordHeader{
		{
			Key:   []byte("ttl"),
			Value: []byte(fmt.Sprintf("%d", int64(ttl.Seconds()))),
		},
		{
			Key:   []byte("created_at"),
			Value: []byte(fmt.Sprintf("%d", time.Now().Unix())),
		},
	}

	msg := &sarama.ProducerMessage{
		Topic:   topicName,
		Key:     sarama.StringEncoder(key),
		Value:   sarama.ByteEncoder(valueBytes),
		Headers: headers,
	}

	_, _, err := s.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (s *Storage) Keys(ctx context.Context, prefix string) ([]string, error) {
	topicName := "xmsgbus_storage"

	if err := s.ensureTopicExists(topicName); err != nil {
		return nil, fmt.Errorf("failed to ensure topic exists: %w", err)
	}

	consumer, err := sarama.NewConsumer(s.brokers, s.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}
	defer consumer.Close()

	partitions, err := consumer.Partitions(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions: %w", err)
	}

	latestMessages := make(map[string]*sarama.ConsumerMessage)
	now := time.Now()

	for _, partition := range partitions {
		partitionConsumer, err := consumer.ConsumePartition(topicName, partition, sarama.OffsetOldest)
		if err != nil {
			continue
		}

		timeout := time.After(5 * time.Second)

	consumeLoop:
		for {
			select {
			case msg := <-partitionConsumer.Messages():
				if msg == nil {
					break consumeLoop
				}

				key := string(msg.Key)
				if !strings.HasPrefix(key, prefix) {
					continue
				}

				if msg.Value == nil {
					delete(latestMessages, key)
				} else {
					latestMessages[key] = msg
				}

			case <-timeout:
				break consumeLoop
			case <-ctx.Done():
				break consumeLoop
			}
		}

		partitionConsumer.Close()
	}

	var keys []string
	for key, msg := range latestMessages {
		if !s.isExpired(msg, now) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (s *Storage) Del(ctx context.Context, key string) error {
	topicName := "xmsgbus_storage"

	if err := s.ensureTopicExists(topicName); err != nil {
		return fmt.Errorf("failed to ensure topic exists: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topicName,
		Key:   sarama.StringEncoder(key),
		Value: nil,
	}

	_, _, err := s.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send delete message: %w", err)
	}

	return nil
}

func (s *Storage) ensureTopicExists(topicName string) error {
	topics, err := s.admin.ListTopics()
	if err != nil {
		return fmt.Errorf("failed to list topics: %w", err)
	}

	if _, exists := topics[topicName]; exists {
		return nil
	}

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
		ConfigEntries: map[string]*string{
			"cleanup.policy": stringPtr("compact"),
		},
	}

	err = s.admin.CreateTopic(topicName, topicDetail, false)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	return nil
}

func (s *Storage) isExpired(msg *sarama.ConsumerMessage, now time.Time) bool {
	var ttlSeconds int64
	var createdAt int64

	for _, header := range msg.Headers {
		switch string(header.Key) {
		case "ttl":
			fmt.Sscanf(string(header.Value), "%d", &ttlSeconds)
		case "created_at":
			fmt.Sscanf(string(header.Value), "%d", &createdAt)
		}
	}

	if ttlSeconds == 0 || createdAt == 0 {
		return false
	}

	expireTime := time.Unix(createdAt, 0).Add(time.Duration(ttlSeconds) * time.Second)
	return now.After(expireTime)
}

func (s *Storage) Close() error {
	var errs []string

	if err := s.producer.Close(); err != nil {
		errs = append(errs, fmt.Sprintf("producer close error: %v", err))
	}

	if err := s.admin.Close(); err != nil {
		errs = append(errs, fmt.Sprintf("admin close error: %v", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}
