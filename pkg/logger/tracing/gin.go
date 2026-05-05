package tracing

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	TraceIDKey = "trace_id"
	SpanIDKey  = "span_id"
)

func GinMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.URL.Path
		}

		ctx, span := otel.Tracer(serviceName).Start(ctx, spanName,
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.route", spanName),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		c.Header(TraceIDKey, span.SpanContext().TraceID().String())
		c.Header(SpanIDKey, span.SpanContext().SpanID().String())

		c.Next()
	}
}

func TraceFromContext(ctx context.Context) (traceID, spanID string) {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		traceID = spanCtx.TraceID().String()
		spanID = spanCtx.SpanID().String()
	}
	return
}

func InjectTraceToContext(ctx context.Context) context.Context {
	propagator := otel.GetTextMapPropagator()
	carrier := make(propagation.HeaderCarrier)
	propagator.Inject(ctx, carrier)
	return ctx
}
