package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/ccheers/xpkg/xmsgbus"
)

type msgBusOptions struct {
	brokers        []string
	groupPrefix    string
	sessionTimeout time.Duration
	config         *sarama.Config
}

func defaultMsgBusOptions() msgBusOptions {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Group.Session.Timeout = DefaultSessionTimeout
	config.Consumer.Group.Heartbeat.Interval = DefaultSessionTimeout / 3
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true
	config.Metadata.AllowAutoTopicCreation = true

	return msgBusOptions{
		brokers:        []string{"localhost:9092"},
		groupPrefix:    DefaultGroupPrefix,
		sessionTimeout: DefaultSessionTimeout,
		config:         config,
	}
}

type IMsgBusOption interface {
	apply(*msgBusOptions)
}

type MsgBusOptionFunc func(*msgBusOptions)

func (fn MsgBusOptionFunc) apply(options *msgBusOptions) {
	fn(options)
}

func WithBrokers(brokers []string) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.brokers = brokers
	}
}

func WithGroupPrefix(prefix string) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.groupPrefix = prefix
	}
}

func WithSessionTimeout(timeout time.Duration) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.sessionTimeout = timeout
		options.config.Consumer.Group.Session.Timeout = timeout
		options.config.Consumer.Group.Heartbeat.Interval = timeout / 3
	}
}

func WithSaramaConfig(config *sarama.Config) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.config = config
	}
}

// WithSASLPlainAuth configures SASL/PLAIN authentication
func WithSASLPlainAuth(username, password string) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.config.Net.SASL.Enable = true
		options.config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		options.config.Net.SASL.User = username
		options.config.Net.SASL.Password = password
	}
}

// WithSASLSCRAMSHA256Auth configures SASL/SCRAM-SHA256 authentication
func WithSASLSCRAMSHA256Auth(username, password string) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.config.Net.SASL.Enable = true
		options.config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		options.config.Net.SASL.User = username
		options.config.Net.SASL.Password = password
		// Note: SCRAMClientGeneratorFunc should be set if using external SCRAM library
		// For now, we rely on sarama's built-in SCRAM support
	}
}

// WithSASLSCRAMSHA512Auth configures SASL/SCRAM-SHA512 authentication
func WithSASLSCRAMSHA512Auth(username, password string) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.config.Net.SASL.Enable = true
		options.config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		options.config.Net.SASL.User = username
		options.config.Net.SASL.Password = password
		// Note: SCRAMClientGeneratorFunc should be set if using external SCRAM library
		// For now, we rely on sarama's built-in SCRAM support
	}
}

// WithTLS enables TLS encryption
func WithTLS(tlsConfig *tls.Config) MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.config.Net.TLS.Enable = true
		if tlsConfig != nil {
			options.config.Net.TLS.Config = tlsConfig
		} else {
			options.config.Net.TLS.Config = &tls.Config{
				InsecureSkipVerify: false,
			}
		}
	}
}

// WithInsecureTLS enables TLS with insecure verification (for testing)
func WithInsecureTLS() MsgBusOptionFunc {
	return func(options *msgBusOptions) {
		options.config.Net.TLS.Enable = true
		options.config.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
}

type MsgBus struct {
	opts     msgBusOptions
	producer sarama.SyncProducer
	admin    sarama.ClusterAdmin
	mu       sync.RWMutex
	channels map[string]map[string]*Consumer
}

type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	cancel        context.CancelFunc
	messageChan   chan *sarama.ConsumerMessage
	topic         string
	channel       string
}

func NewMsgBus(options ...IMsgBusOption) (xmsgbus.IMsgBus, error) {
	opts := defaultMsgBusOptions()
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

	return &MsgBus{
		opts:     opts,
		producer: producer,
		admin:    admin,
		channels: make(map[string]map[string]*Consumer),
	}, nil
}

func (x *MsgBus) Push(ctx context.Context, topic string, bs []byte) error {
	if err := x.ensureTopicExists(topic); err != nil {
		return fmt.Errorf("failed to ensure topic exists: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(bs),
	}

	_, _, err := x.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", topic, err)
	}

	return nil
}

func (x *MsgBus) Pop(ctx context.Context, topic, channel string, blockTimeout time.Duration) ([]byte, func(), error) {
	x.mu.RLock()
	consumer := x.getConsumer(topic, channel)
	x.mu.RUnlock()

	if consumer == nil {
		return nil, nil, fmt.Errorf("consumer not found for topic %s, channel %s", topic, channel)
	}

	if blockTimeout > 0 {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(blockTimeout):
			return nil, nil, xmsgbus.ErrPopTimeout
		case msg := <-consumer.messageChan:
			return msg.Value, func() {}, nil
		}
	}

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case msg := <-consumer.messageChan:
		return msg.Value, func() {}, nil
	}
}

func (x *MsgBus) AddChannel(ctx context.Context, topic string, channel string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if err := x.ensureTopicExists(topic); err != nil {
		return fmt.Errorf("failed to ensure topic exists: %w", err)
	}

	if x.channels[topic] == nil {
		x.channels[topic] = make(map[string]*Consumer)
	}

	if x.channels[topic][channel] != nil {
		return nil
	}

	groupID := fmt.Sprintf("%s_%s_%s", x.opts.groupPrefix, topic, channel)
	consumerGroup, err := sarama.NewConsumerGroup(x.opts.brokers, groupID, x.opts.config)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	consumer := &Consumer{
		consumerGroup: consumerGroup,
		cancel:        cancel,
		messageChan:   make(chan *sarama.ConsumerMessage, 100),
		topic:         topic,
		channel:       channel,
	}

	x.channels[topic][channel] = consumer

	go func() {
		defer func() {
			close(consumer.messageChan)
			consumerGroup.Close()
		}()

		handler := &consumerGroupHandler{messageChan: consumer.messageChan}
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := consumerGroup.Consume(ctx, []string{topic}, handler); err != nil {
					time.Sleep(time.Second)
					continue
				}
			}
		}
	}()

	return nil
}

func (x *MsgBus) RemoveChannel(ctx context.Context, topic string, channel string) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.channels[topic] == nil {
		return nil
	}

	consumer := x.channels[topic][channel]
	if consumer == nil {
		return nil
	}

	consumer.cancel()
	delete(x.channels[topic], channel)

	if len(x.channels[topic]) == 0 {
		delete(x.channels, topic)
	}

	return nil
}

func (x *MsgBus) ListChannel(ctx context.Context, topic string) ([]string, error) {
	x.mu.RLock()
	defer x.mu.RUnlock()

	if x.channels[topic] == nil {
		return nil, nil
	}

	channels := make([]string, 0, len(x.channels[topic]))
	for channel := range x.channels[topic] {
		channels = append(channels, channel)
	}

	return channels, nil
}

func (x *MsgBus) getConsumer(topic, channel string) *Consumer {
	if x.channels[topic] == nil {
		return nil
	}
	return x.channels[topic][channel]
}

func (x *MsgBus) Close() error {
	x.mu.Lock()
	defer x.mu.Unlock()

	var errs []string

	for topic, channels := range x.channels {
		for channel, consumer := range channels {
			consumer.cancel()
			delete(channels, channel)
		}
		delete(x.channels, topic)
	}

	if err := x.producer.Close(); err != nil {
		errs = append(errs, fmt.Sprintf("producer close error: %v", err))
	}

	if err := x.admin.Close(); err != nil {
		errs = append(errs, fmt.Sprintf("admin close error: %v", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

type consumerGroupHandler struct {
	messageChan chan<- *sarama.ConsumerMessage
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		select {
		case h.messageChan <- message:
			session.MarkMessage(message, "")
		case <-session.Context().Done():
			return nil
		}
	}
	return nil
}

func (x *MsgBus) ensureTopicExists(topicName string) error {
	topics, err := x.admin.ListTopics()
	if err != nil {
		return fmt.Errorf("failed to list topics: %w", err)
	}

	if _, exists := topics[topicName]; exists {
		return nil
	}

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 2,
		ConfigEntries: map[string]*string{
			"cleanup.policy": stringPtr("delete"),
			"retention.ms":   stringPtr("7200000"), // 2 hours = 2 * 60 * 60 * 1000 ms
		},
	}

	err = x.admin.CreateTopic(topicName, topicDetail, false)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	return nil
}
