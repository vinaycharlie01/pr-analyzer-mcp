package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Provider struct {
	tracer        trace.Tracer
	meter         metric.Meter
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

type Config struct {
	ServiceName string
	Enabled     bool
}

func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{
			tracer: otel.Tracer(cfg.ServiceName),
			meter:  noop.NewMeterProvider().Meter(cfg.ServiceName),
		}, nil
	}

	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("creating trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(tp)

	// Use a noop meter provider — wire in a real exporter (e.g. OTLP) when needed.
	mp := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(mp)

	return &Provider{
		tracer:         tp.Tracer(cfg.ServiceName),
		meter:          mp.Meter(cfg.ServiceName),
		tracerProvider: tp,
		meterProvider:  mp,
	}, nil
}

func (p *Provider) Tracer() trace.Tracer {
	return p.tracer
}

func (p *Provider) Meter() metric.Meter {
	return p.meter
}

func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tracerProvider != nil {
		if err := p.tracerProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutting down tracer provider: %w", err)
		}
	}
	if p.meterProvider != nil {
		if err := p.meterProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutting down meter provider: %w", err)
		}
	}
	return nil
}
