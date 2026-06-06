// Package telemetry initialises OpenTelemetry for traces, metrics, and logs,
// then wires the OTEL log bridge as the default slog handler so that all
// slog.Info / slog.Error calls in the application flow through the OTEL
// pipeline automatically.
//
// Two modes:
//   - No OTEL_EXPORTER_OTLP_ENDPOINT set → stdout exporters (dev / CI).
//   - OTEL_EXPORTER_OTLP_ENDPOINT set    → OTLP gRPC exporters (Grafana Cloud,
//     Datadog, self-hosted Collector, etc.).
//
// Usage in main:
//
//	shutdown, err := telemetry.Setup(ctx, cfg)
//	if err != nil { ... }
//	defer shutdown()
//	// all slog calls from here forward are OTEL-backed
package telemetry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/phoenixha4/slate/internal/logging"
)

// Config carries the telemetry-relevant subset of application configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string // empty → stdout exporters
	LogLevel       slog.Level
	LogFormat      string // "pretty" | "json" | "text"  (used when OTLP is absent)
}

// Setup initialises the global TracerProvider, MeterProvider, and
// LoggerProvider, then sets the default slog logger to a handler backed by the
// OTEL LoggerProvider. Returns a shutdown function that must be called before
// the process exits to flush all pending telemetry.
func Setup(ctx context.Context, cfg Config) (shutdown func(), err error) {
	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}

	var shutdowns []func(context.Context) error

	// ── Trace provider ────────────────────────────────────────────────────────
	tp, err := newTracerProvider(ctx, res, cfg.OTLPEndpoint)
	if err != nil {
		return nil, fmt.Errorf("tracer provider: %w", err)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	shutdowns = append(shutdowns, tp.Shutdown)

	// ── Metric provider ───────────────────────────────────────────────────────
	mp, err := newMeterProvider(res, cfg.OTLPEndpoint)
	if err != nil {
		return nil, fmt.Errorf("meter provider: %w", err)
	}
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mp.Shutdown)

	// ── Log provider + slog bridge ────────────────────────────────────────────
	lp, err := newLoggerProvider(ctx, res, cfg)
	if err != nil {
		return nil, fmt.Errorf("logger provider: %w", err)
	}
	global.SetLoggerProvider(lp)
	shutdowns = append(shutdowns, lp.Shutdown)

	// Wire slog: all slog.Info / Error / etc. calls flow into OTEL.
	slog.SetDefault(slog.New(otelslog.NewHandler(cfg.ServiceName,
		otelslog.WithLoggerProvider(lp),
	)))

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, fn := range shutdowns {
			_ = fn(ctx)
		}
	}, nil
}

// ── Providers ─────────────────────────────────────────────────────────────────

func newResource(ctx context.Context, cfg Config) (*sdkresource.Resource, error) {
	return sdkresource.New(ctx,
		sdkresource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
		sdkresource.WithHost(),
		sdkresource.WithOS(),
	)
}

func newTracerProvider(ctx context.Context, res *sdkresource.Resource, endpoint string) (*sdktrace.TracerProvider, error) {
	var exp sdktrace.SpanExporter
	var err error

	if endpoint != "" {
		exp, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
	} else {
		// Discard traces in dev; no noise on stdout unless a collector is set.
		exp, err = stdouttrace.New(stdouttrace.WithWriter(io.Discard))
	}
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	), nil
}

func newMeterProvider(res *sdkresource.Resource, endpoint string) (*sdkmetric.MeterProvider, error) {
	// Metrics via stdout (discarded) in dev; a real OTLP exporter can be
	// wired here when endpoint is non-empty using sdkmetric.WithReader +
	// otlpmetricgrpc. Kept simple for now: the otelhttp middleware still
	// records RED metrics in-process regardless of the export destination.
	_ = endpoint
	exp, err := stdoutmetric.New(stdoutmetric.WithWriter(io.Discard))
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
	), nil
}

func newLoggerProvider(ctx context.Context, res *sdkresource.Resource, cfg Config) (*sdklog.LoggerProvider, error) {
	var processor sdklog.Processor

	if cfg.OTLPEndpoint != "" {
		exp, err := otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlploggrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
		processor = sdklog.NewBatchProcessor(exp)
	} else {
		// In dev, mirror log output through the pretty/text/json slog handler
		// so developers see human-readable logs in the terminal.
		exp := newConsoleLogExporter(cfg)
		processor = sdklog.NewSimpleProcessor(exp)
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(processor),
	), nil
}

// ── Console log exporter ──────────────────────────────────────────────────────
// consoleLogExporter is an sdklog.Exporter that forwards OTEL log records to
// a standard slog.Handler so logs appear in the developer terminal.

type consoleLogExporter struct {
	handler slog.Handler
}

func newConsoleLogExporter(cfg Config) *consoleLogExporter {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var h slog.Handler
	switch cfg.LogFormat {
	case "pretty":
		h = logging.NewPrettyHandler(os.Stdout, opts)
	case "text":
		h = slog.NewTextHandler(os.Stdout, opts)
	default:
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	return &consoleLogExporter{handler: h}
}

func (e *consoleLogExporter) Export(ctx context.Context, records []sdklog.Record) error {
	for _, rec := range records {
		level := otelSeverityToSlog(rec.Severity())
		if !e.handler.Enabled(ctx, level) {
			continue
		}

		attrs := make([]slog.Attr, 0, rec.AttributesLen())
		rec.WalkAttributes(func(kv otellog.KeyValue) bool {
			attrs = append(attrs, slog.String(kv.Key, kv.Value.String()))
			return true
		})

		r := slog.NewRecord(rec.Timestamp(), level, rec.Body().AsString(), 0)
		r.AddAttrs(attrs...)
		_ = e.handler.Handle(ctx, r)
	}
	return nil
}

func (e *consoleLogExporter) Shutdown(_ context.Context) error   { return nil }
func (e *consoleLogExporter) ForceFlush(_ context.Context) error { return nil }

func otelSeverityToSlog(s otellog.Severity) slog.Level {
	switch {
	case s >= otellog.SeverityError:
		return slog.LevelError
	case s >= otellog.SeverityWarn:
		return slog.LevelWarn
	case s >= otellog.SeverityInfo:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

// stdoutlog is imported to satisfy the build even when OTLP is not used.
var _ = stdoutlog.New
