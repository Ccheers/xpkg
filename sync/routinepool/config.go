package routinepool

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const (
	defaultScalaThreshold = 1
)

// Config is used to config pool.
type Config struct {
	// threshold for scale.
	// new goroutine is created if len(task chan) > ScaleThreshold.
	// defaults to defaultScalaThreshold.
	scaleThreshold int32

	// metrics provider.
	// defaults to otel.GetMeterProvider().
	meterProvider metric.MeterProvider
	// This method will be called when the task is canceled.
	errorHandler func(context.Context, error)
	// This method will be called when the worker panic.
	panicHandler func(context.Context, error)
}

type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (f optionFunc) apply(config *Config) {
	f(config)
}

// WithScaleThreshold sets the scale threshold.
func WithScaleThreshold(threshold int32) Option {
	return optionFunc(func(config *Config) {
		config.scaleThreshold = threshold
	})
}

// WithMeterProvider sets the meter provider.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return optionFunc(func(config *Config) {
		config.meterProvider = mp
	})
}

// WithErrorHandler sets the panic handler.
func WithErrorHandler(f func(context.Context, error)) Option {
	return optionFunc(func(config *Config) {
		config.errorHandler = f
	})
}

// WithErrorHandler sets the panic handler.
func WithPanicHandler(f func(context.Context, error)) Option {
	return optionFunc(func(config *Config) {
		config.panicHandler = f
	})
}

// NewConfig creates a default Config.
func NewConfig(opts ...Option) *Config {
	c := &Config{
		scaleThreshold: defaultScalaThreshold,
		meterProvider:  otel.GetMeterProvider(),
		errorHandler: func(ctx context.Context, err error) {
			log.Printf("[ERROR]: %v\n", err)
		},
		panicHandler: func(ctx context.Context, err error) {
			log.Printf("[PANIC]: %v\n", err)
		},
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}
