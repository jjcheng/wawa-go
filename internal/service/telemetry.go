package service

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type Telemetry struct {
	tracerProvider    *sdktrace.TracerProvider
	appInsightsClient appinsights.TelemetryClient
	shutdown          func(context.Context) error
}

func NewTelemetry() *Telemetry {
	//connectionString := cfg.Default().Site.AzureAppinsightsConnectionString
	connectionString := ""
	if connectionString == "" {
		return nil
	}
	// Extract instrumentation key from connection string
	instrumentationKey := extractInstrumentationKey(connectionString)
	if instrumentationKey == "" {
		//panic("Could not extract instrumentation key from connection string")
	}
	// Create Application Insights client
	appInsightsClient := appinsights.NewTelemetryClient(instrumentationKey)
	// Configure the telemetry client
	appInsightsClient.Context().Tags.Cloud().SetRole("ai-go-app")
	appInsightsClient.Context().Tags.Cloud().SetRoleInstance(getCurrentInstanceName())
	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("ai-go-app"),
			semconv.ServiceVersion(cfg.Default().Site.Version),
			semconv.DeploymentEnvironment(string(cfg.Default().Site.Environment)),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("Error creating resource: %v", err.Error()))
	}
	// Use custom Application Insights exporter
	exporter := &appInsightsExporter{
		client: appInsightsClient,
	}
	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	// Set global tracer provider
	otel.SetTracerProvider(tp)
	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return &Telemetry{
		tracerProvider:    tp,
		appInsightsClient: appInsightsClient,
		shutdown: func(ctx context.Context) error {
			// Flush any remaining telemetry
			appInsightsClient.Channel().Flush()
			// Wait a bit for flush to complete
			time.Sleep(2 * time.Second)
			return tp.Shutdown(ctx)
		},
	}
}

// Shutdown gracefully shuts down the Telemetry service
func (ots *Telemetry) Shutdown(ctx context.Context) error {
	if ots.shutdown != nil {
		return ots.shutdown(ctx)
	}
	return nil
}

// GetTracer returns a tracer for the given name
func (ots *Telemetry) GetTracer(name string) trace.Tracer {
	if ots.tracerProvider == nil {
		return otel.Tracer(name) // Return no-op tracer if not initialized
	}
	return ots.tracerProvider.Tracer(name)
}

// TrackHTTPRequest sends HTTP request telemetry to Application Insights
func (ots *Telemetry) TrackHTTPRequest(method, path, query, requestJSON, userAgent, remoteAddress, requestBody string, userId int32, responseSize int, duration time.Duration, statusCode int, requestID string) {
	requestTelemetry := appinsights.NewRequestTelemetry(method, path, duration, fmt.Sprint(statusCode))
	requestTelemetry.Name = "HTTP REQUEST"
	requestTelemetry.Success = statusCode < 400
	requestTelemetry.Properties["environment"] = string(cfg.Default().Site.Environment)
	requestTelemetry.Properties["userId"] = fmt.Sprint(userId)
	requestTelemetry.Properties["method"] = method
	requestTelemetry.Properties["path"] = path
	requestTelemetry.Properties["query"] = query
	requestTelemetry.Properties["remoteAddress"] = remoteAddress
	requestTelemetry.Properties["userAgent"] = userAgent
	requestTelemetry.Properties["requestBody"] = requestBody
	requestTelemetry.Properties["requestJSON"] = requestJSON
	requestTelemetry.Properties["status"] = fmt.Sprint(statusCode)
	requestTelemetry.Properties["responseSize"] = fmt.Sprint(responseSize)
	requestTelemetry.Properties["requestId"] = requestID
	ots.appInsightsClient.Track(requestTelemetry)
	ots.appInsightsClient.Channel().Flush()
}

// TrackError sends an error to Application Insights
func (ots *Telemetry) TrackError(err error, properties map[string]string) {
	exception := appinsights.NewExceptionTelemetry(err)
	maps.Copy(exception.Properties, properties)
	ots.appInsightsClient.Track(exception)
	ots.appInsightsClient.Channel().Flush()
}

// TrackEvent sends a custom event to Application Insights
func (ots *Telemetry) TrackEvent(name string, properties map[string]string, measurements map[string]float64) {
	event := appinsights.NewEventTelemetry(name)
	maps.Copy(event.Properties, properties)
	maps.Copy(event.Measurements, measurements)
	ots.appInsightsClient.Track(event)
	ots.appInsightsClient.Channel().Flush()
}

// TrackDependency tracks external dependencies
func (ots *Telemetry) TrackDependency(name, dependencyType, target string, success bool, duration time.Duration) {
	dependency := appinsights.NewRemoteDependencyTelemetry(name, dependencyType, target, success)
	dependency.Duration = duration
	ots.appInsightsClient.Track(dependency)
	ots.appInsightsClient.Channel().Flush()
}

// TrackTrace sends a trace message to Application Insights
func (ots *Telemetry) TrackTrace(message string, severity string, properties map[string]string) {
	ots.appInsightsClient.TrackTrace(message, appinsights.Information)
}

// Custom exporter for Application Insights
type appInsightsExporter struct {
	client appinsights.TelemetryClient
}

func (e *appInsightsExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		// Convert Telemetry span to Application Insights telemetry
		telemetry := appinsights.NewRequestTelemetry(
			"HTTP",
			span.Name(),
			span.EndTime().Sub(span.StartTime()),
			"200", // Default success code
		)
		// Add span attributes as properties
		for _, attr := range span.Attributes() {
			telemetry.Properties[string(attr.Key)] = attr.Value.AsString()
		}
		// Set success based on span status
		telemetry.Success = span.Status().Code != codes.Error
		if span.Status().Code == codes.Error {
			telemetry.ResponseCode = "500"
		}
		// Set trace ID and span ID
		telemetry.Id = span.SpanContext().SpanID().String()
		telemetry.Properties["traceId"] = span.SpanContext().TraceID().String()
		e.client.Track(telemetry)
	}
	e.client.Channel().Flush()
	return nil
}

func (e *appInsightsExporter) Shutdown(ctx context.Context) error {
	e.client.Channel().Flush()
	time.Sleep(2 * time.Second)
	return nil
}

func extractInstrumentationKey(connectionString string) string {
	parts := strings.SplitSeq(connectionString, ";")
	for part := range parts {
		if after, ok := strings.CutPrefix(part, "InstrumentationKey="); ok {
			return after
		}
	}
	panic("Unable to get InstrumentationKey")
}

// getCurrentInstanceName generates a dynamic instance name for the current running instance
func getCurrentInstanceName() string {
	// Try to get hostname first
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}
	// Get process ID for uniqueness
	pid := os.Getpid()
	// Check for container ID (first 12 chars of hostname in containers)
	if len(hostname) >= 12 {
		// Likely running in a container, use hostname as is
		return hostname
	}
	// Check for environment variable override
	if instanceName := os.Getenv("INSTANCE_NAME"); instanceName != "" {
		return instanceName
	}
	// Fallback: combine hostname with PID
	return hostname + "-" + strconv.Itoa(pid)
}
