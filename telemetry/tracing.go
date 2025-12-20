// Package telemetry provides observability utilities.
package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig holds tracing configuration.
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string  // OTLP endpoint
	SampleRate     float64 // 0.0 to 1.0
	Insecure       bool    // Use insecure connection
}

// DefaultTracingConfig returns default configuration.
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		SampleRate: 1.0, // Sample everything by default
	}
}

// TracingProvider provides tracing functionality.
type TracingProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	config   TracingConfig
}

// NewTracingProvider creates a new tracing provider.
func NewTracingProvider(ctx context.Context, config TracingConfig) (*TracingProvider, error) {
	// Create OTLP exporter
	opts := []otlptracehttp.Option{}

	if config.Endpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(config.Endpoint))
	}

	if config.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if config.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SampleRate)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global provider and propagator
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Get tracer
	tracer := provider.Tracer(config.ServiceName)

	return &TracingProvider{
		provider: provider,
		tracer:   tracer,
		config:   config,
	}, nil
}

// Tracer returns the tracer for creating spans.
func (t *TracingProvider) Tracer() trace.Tracer {
	return t.tracer
}

// Shutdown shuts down the tracing provider.
func (t *TracingProvider) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

// StartSpan starts a new span.
func (t *TracingProvider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// Span utilities

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceID returns the trace ID from context.
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// SpanID returns the span ID from context.
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// AddSpanEvent adds an event to the current span.
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanError records an error on the current span.
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanAttributes sets attributes on the current span.
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// Common attribute helpers

// HTTPServerAttributes returns common HTTP server span attributes.
func HTTPServerAttributes(r *http.Request, statusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPMethod(r.Method),
		semconv.HTTPURL(r.URL.String()),
		semconv.HTTPRoute(r.URL.Path),
		semconv.HTTPStatusCode(statusCode),
		semconv.HTTPScheme(r.URL.Scheme),
		attribute.String("client.address", r.RemoteAddr),
		semconv.UserAgentOriginal(r.UserAgent()),
	}
}

// HTTPClientAttributes returns common HTTP client span attributes.
func HTTPClientAttributes(method, url string, statusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.HTTPMethod(method),
		semconv.HTTPURL(url),
		semconv.HTTPStatusCode(statusCode),
	}
}

// DatabaseAttributes returns common database span attributes.
func DatabaseAttributes(dbType, operation, table string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.DBSystemKey.String(dbType),
		semconv.DBOperation(operation),
		semconv.DBSQLTable(table),
	}
}

// MessagingAttributes returns common messaging span attributes.
func MessagingAttributes(system, destination, operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("messaging.system", system),
		semconv.MessagingDestinationName(destination),
		attribute.String("messaging.operation.name", operation),
	}
}

// Rideshare-specific attributes

// TripAttributes returns trip-related span attributes.
func TripAttributes(tripID, riderID, driverID, status string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("trip.id", tripID),
		attribute.String("trip.status", status),
	}
	if riderID != "" {
		attrs = append(attrs, attribute.String("trip.rider_id", riderID))
	}
	if driverID != "" {
		attrs = append(attrs, attribute.String("trip.driver_id", driverID))
	}
	return attrs
}

// DriverAttributes returns driver-related span attributes.
func DriverAttributes(driverID string, lat, lng float64) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("driver.id", driverID),
		attribute.Float64("driver.latitude", lat),
		attribute.Float64("driver.longitude", lng),
	}
}

// PaymentAttributes returns payment-related span attributes.
func PaymentAttributes(paymentID, method, status string, amount float64) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("payment.id", paymentID),
		attribute.String("payment.method", method),
		attribute.String("payment.status", status),
		attribute.Float64("payment.amount", amount),
	}
}

// TracingMiddleware creates an HTTP middleware that adds tracing.
func TracingMiddleware(tracer trace.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from incoming request
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start span
			spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPMethod(r.Method),
					semconv.HTTPURL(r.URL.String()),
					semconv.HTTPRoute(r.URL.Path),
					attribute.String("client.address", r.RemoteAddr),
					semconv.UserAgentOriginal(r.UserAgent()),
				),
			)
			defer span.End()

			// Wrap response writer to capture status code
			wrapped := &tracingResponseWriter{ResponseWriter: w, status: http.StatusOK}

			// Continue with request
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Record status
			span.SetAttributes(semconv.HTTPStatusCode(wrapped.status))

			// Set span status based on HTTP status
			if wrapped.status >= 400 {
				span.SetStatus(codes.Error, http.StatusText(wrapped.status))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}

type tracingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *tracingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// InjectTraceContext injects trace context into outgoing HTTP request.
func InjectTraceContext(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// HTTPClientTracer creates a traced HTTP client.
type HTTPClientTracer struct {
	client *http.Client
	tracer trace.Tracer
}

// NewHTTPClientTracer creates a new traced HTTP client.
func NewHTTPClientTracer(tracer trace.Tracer) *HTTPClientTracer {
	return &HTTPClientTracer{
		client: &http.Client{},
		tracer: tracer,
	}
}

// Do performs an HTTP request with tracing.
func (c *HTTPClientTracer) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	spanName := fmt.Sprintf("HTTP %s %s", req.Method, req.URL.Host)
	ctx, span := c.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPMethod(req.Method),
			semconv.HTTPURL(req.URL.String()),
		),
	)
	defer span.End()

	// Inject trace context
	InjectTraceContext(ctx, req)

	// Perform request
	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(semconv.HTTPStatusCode(resp.StatusCode))

	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return resp, nil
}

// WrapDatabaseOperation wraps a database operation with tracing.
func WrapDatabaseOperation(ctx context.Context, tracer trace.Tracer, dbType, operation, table string, fn func(context.Context) error) error {
	spanName := fmt.Sprintf("%s %s", operation, table)
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(DatabaseAttributes(dbType, operation, table)...),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// WrapMessagingOperation wraps a messaging operation with tracing.
func WrapMessagingOperation(ctx context.Context, tracer trace.Tracer, system, destination, operation string, fn func(context.Context) error) error {
	spanName := fmt.Sprintf("%s %s", operation, destination)
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(MessagingAttributes(system, destination, operation)...),
	)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}
