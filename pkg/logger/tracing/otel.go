package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type TracingProvider struct {
	tp *sdktrace.TracerProvider
}

type TracingConfig struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
	SampleRate  float64
}

func NewTracingProvider(ctx context.Context, cfg TracingConfig) (*TracingProvider, error) {
	if !cfg.Enabled {
		return &TracingProvider{tp: nil}, nil
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(time.Second)),
	)

	return &TracingProvider{tp: tp}, nil
}

func (t *TracingProvider) Shutdown(ctx context.Context) error {
	if t.tp == nil {
		return nil
	}
	return t.tp.Shutdown(ctx)
}

func InitGlobalTracer(ctx context.Context, cfg TracingConfig) error {
	if !cfg.Enabled {
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return nil
	}

	tp, err := NewTracingProvider(ctx, cfg)
	if err != nil {
		return err
	}

	otel.SetTracerProvider(tp.tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

func NewTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
