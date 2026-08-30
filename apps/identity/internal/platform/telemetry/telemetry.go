package telemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Setup configures OTLP only when an endpoint is explicitly provided. A
// missing collector therefore degrades telemetry without breaking identity
// issuance or request handling.
func Setup(ctx context.Context, endpoint, environment string, insecure bool) (func(context.Context) error, error) {
	if strings.TrimSpace(endpoint) == "" {
		return func(context.Context) error { return nil }, nil
	}
	options := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}
	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, err
	}
	serviceResource, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName("identity"), semconv.DeploymentEnvironment(environment)),
	)
	if err != nil {
		return nil, err
	}
	provider := trace.NewTracerProvider(trace.WithBatcher(exporter), trace.WithResource(serviceResource))
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}
