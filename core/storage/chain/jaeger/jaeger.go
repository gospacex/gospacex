package jaeger

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gospacex/gospacex/core/storage/conf"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// initTracer 初始化 Tracer
func Init(cfg *conf.ChainConfig) (err error) {
	_, err = initTracer(context.Background(), cfg.Host, cfg.Port, cfg.Name)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func initTracer(ctx context.Context, host string, port int, serviceName string) (*sdktrace.TracerProvider, error) {
	tp, err := newJaegerTraceProvider(ctx, host, port, serviceName)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	)
	return tp, nil
}

func newJaegerTraceProvider(ctx context.Context, host string, port int, serviceName string) (*sdktrace.TracerProvider, error) {
	// 创建一个使用 HTTP 协议连接本机Jaeger的 Exporter
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%s", host, strconv.Itoa(port))),
		otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, err
	}
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 采样
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(time.Second)),
	)
	return traceProvider, nil
}
