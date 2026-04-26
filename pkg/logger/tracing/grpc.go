package tracing

import (
	"context"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	GRPCTraceIDKey = "trace_id"
	GRPCSpanIDKey  = "span_id"
)

func GRPCServerInterceptor() grpc.ServerOption {
	return grpc.StatsHandler(otelgrpc.NewServerHandler(
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
	))
}

func GRPCClientInterceptor() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler(
		otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
	))
}

func GRPCServerTraceMiddleware(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		propagator := otel.GetTextMapPropagator()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(md))

		ctx, span := otel.Tracer(serviceName).Start(ctx, info.FullMethod,
			trace.WithAttributes(
				attribute.String("grpc.method", info.FullMethod),
				attribute.String("grpc.service", info.FullMethod),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		return handler(ctx, req)
	}
}

func GRPCClientTraceUnaryInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		propagator := otel.GetTextMapPropagator()
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		carrier := propagation.HeaderCarrier(md)
		propagator.Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, md)

		ctx, span := otel.Tracer(serviceName).Start(ctx, method,
			trace.WithAttributes(
				attribute.String("grpc.method", method),
			),
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func GRPCClientTraceStreamInterceptor(serviceName string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		propagator := otel.GetTextMapPropagator()
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		carrier := propagation.HeaderCarrier(md)
		propagator.Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, md)

		ctx, span := otel.Tracer(serviceName).Start(ctx, method,
			trace.WithAttributes(
				attribute.String("grpc.method", method),
			),
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		return streamer(ctx, desc, cc, method, opts...)
	}
}

func GetTraceIDFromGRPCContext(ctx context.Context) (traceID, spanID string) {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String(), spanCtx.SpanID().String()
	}
	return
}
