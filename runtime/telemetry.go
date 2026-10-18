package runtime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thruputchain/chuckd"
	"github.com/thruputchain/chuckd/util/blocktrace"
	"github.com/thruputchain/lib-go/liblog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"
)

// PprofConfig controls live profiling and endpoints reserved by file profiling.
type PprofConfig struct {
	Enabled            bool
	AllowRemote        bool
	CPUProfileActive   bool
	TraceProfileActive bool
}

// SetupTelemetry initializes OpenTelemetry with Prometheus and returns a function to shut it down.
func SetupTelemetry(ctx context.Context, instanceID string, addr string, config PprofConfig) (shutdown func(context.Context) error, err error) {
	t, err := startTelemetry(ctx, instanceID, addr, config)
	if err != nil {
		return nil, err
	}
	return t.shutdown, nil
}

const maxConcurrentPprof = 2
const maxPprofSeconds = 60
const telemetryReadHeaderTimeout = 10 * time.Second
const telemetryIdleTimeout = 60 * time.Second

type telemetry struct {
	addr net.Addr
	srv  *http.Server
	mp   *sdkmetric.MeterProvider
	tp   *sdktrace.TracerProvider
}

// shutdown closes the metrics server before flushing providers, so an in-flight
// profile cannot delay exit.
func (t *telemetry) shutdown(ctx context.Context) error {
	return errors.Join(t.srv.Close(), newShutdown(t.tp, t.mp)(ctx))
}

const traceShutdownTimeout = 5 * time.Second

func newShutdown(tracerProvider *sdktrace.TracerProvider, meterProvider *sdkmetric.MeterProvider) func(context.Context) error {
	return func(ctx context.Context) error {
		var traceErr error
		if tracerProvider != nil {
			traceCtx, cancel := context.WithTimeout(ctx, traceShutdownTimeout)
			defer cancel()
			traceErr = tracerProvider.Shutdown(traceCtx)
		}
		var metricErr error
		if meterProvider != nil {
			metricErr = meterProvider.Shutdown(ctx)
		}
		return errors.Join(traceErr, metricErr)
	}
}

func isLoopback(ip net.IP) bool { return ip.IsLoopback() }

func pprofRemoteAllowed(config PprofConfig) bool {
	return config.AllowRemote || os.Getenv("GROUNDHOG_PPROF_ALLOW_REMOTE") == "1"
}

func newTelemetryServer(mux http.Handler) *http.Server {
	return &http.Server{Handler: mux, ReadHeaderTimeout: telemetryReadHeaderTimeout, IdleTimeout: telemetryIdleTimeout}
}

func startTelemetry(ctx context.Context, instanceID, addr string, config PprofConfig) (*telemetry, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("binding metrics listener: %w", err)
	}

	rsc, err := createResource(instanceID)
	if err != nil {
		ln.Close()
		return nil, err
	}

	// Create a Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		ln.Close()
		return nil, err
	}

	// Create a meter provider with the Prometheus exporter
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(rsc),
	)

	// Set the global meter provider
	otel.SetMeterProvider(meterProvider)
	tracerProvider := setupTracing(ctx, instanceID)

	// Set up HTTP server to expose Prometheus metrics endpoint
	remote := !isLoopback(ln.Addr().(*net.TCPAddr).IP)
	allowRemote := pprofRemoteAllowed(config)
	servePprof := config.Enabled && (!remote || allowRemote)
	if config.Enabled && remote {
		if !allowRemote {
			liblog.Warnf(ctx, "pprof disabled: metrics listener %s is not loopback; pass --pprof-allow-remote or set GROUNDHOG_PPROF_ALLOW_REMOTE=1 to serve unauthenticated profiles beyond loopback", ln.Addr())
		} else {
			liblog.Warnf(ctx, "unauthenticated pprof endpoints are reachable beyond loopback on %s", ln.Addr())
		}
	}
	if servePprof {
		liblog.Infof(ctx, "pprof enabled at http://%s/debug/pprof/ (remote access allowed=%t)", ln.Addr(), allowRemote)
	}
	srv := newTelemetryServer(telemetryMux(config, servePprof, allowRemote))
	if servePprof && config.CPUProfileActive {
		liblog.Infof(ctx, "pprof /debug/pprof/profile unavailable: --cpu file profiling is active")
	}
	if servePprof && config.TraceProfileActive {
		liblog.Infof(ctx, "pprof /debug/pprof/trace unavailable: --trace file profiling is active")
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			liblog.Errorf(ctx, "serving metrics: %v", err)
		}
	}()

	// Return a shutdown function that cleans up resources
	liblog.Infof(ctx, "prometheus metrics endpoint started at %s/metrics", ln.Addr())
	return &telemetry{addr: ln.Addr(), srv: srv, mp: meterProvider, tp: tracerProvider}, nil
}

func telemetryMux(config PprofConfig, servePprof, allowRemote bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	if servePprof {
		sem := make(chan struct{}, maxConcurrentPprof)
		handle := func(path string, h http.Handler) {
			mux.Handle(path, guardPprof(allowRemote, limitPprof(sem, h)))
		}
		handle("/debug/pprof/", http.HandlerFunc(pprofIndex))
		if config.CPUProfileActive {
			handle("/debug/pprof/profile", profileConflict("--cpu"))
		} else {
			handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		}
		handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		if config.TraceProfileActive {
			handle("/debug/pprof/trace", profileConflict("--trace"))
		} else {
			handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		}
		for _, name := range []string{"heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
			handle("/debug/pprof/"+name, pprof.Handler(name))
		}
	}
	return mux
}

func guardPprof(allowRemote bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
		ip := net.ParseIP(host)
		// DNS names can rebind to the listener; remote opt-in permits direct IP access only.
		if !strings.EqualFold(host, "localhost") && !(ip != nil && (ip.IsLoopback() || allowRemote)) {
			http.Error(w, "pprof requires a loopback or IP-literal Host header", http.StatusForbidden)
			return
		}
		if _, present := r.Header["Origin"]; present {
			origin, err := url.Parse(r.Header.Get("Origin"))
			if err != nil || !strings.EqualFold(origin.Host, r.Host) {
				http.Error(w, "pprof requires a same-host Origin header", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func limitPprof(sem chan struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allow := "GET, HEAD"
		if r.URL.Path == "/debug/pprof/symbol" {
			allow += ", POST"
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead &&
			!(r.Method == http.MethodPost && r.URL.Path == "/debug/pprof/symbol") {
			w.Header().Set("Allow", allow)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		values := r.URL.Query()["seconds"]
		if r.Method != http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			values = r.Form["seconds"]
		}
		if len(values) > 1 {
			http.Error(w, "seconds must be given at most once", http.StatusBadRequest)
			return
		}
		if len(values) == 1 {
			n, err := strconv.ParseInt(values[0], 10, 64)
			if err != nil || n < 0 || n > maxPprofSeconds {
				http.Error(w, "seconds must be an integer between 0 and 60", http.StatusBadRequest)
				return
			}
		}
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
		default:
			http.Error(w, "too many concurrent pprof requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func pprofIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/debug/pprof/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, "<!doctype html><title>pprof</title><ul>")
	for _, name := range []string{"profile", "trace", "symbol", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
		fmt.Fprintf(w, "<li><a href=\"/debug/pprof/%s\">%s</a></li>\n", name, name)
	}
	fmt.Fprintln(w, "</ul>")
}

func profileConflict(flag string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "endpoint unavailable while "+flag+" file profiling is active", http.StatusConflict)
	}
}

// setupTracing leaves tracing disabled unless the standard OTLP endpoint is set.
func setupTracing(ctx context.Context, instanceID string) *sdktrace.TracerProvider {
	if strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") ||
		os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return nil
	}
	warn := func(err error) {
		liblog.Warnf(ctx, "trace exporter: %v", err)
	}
	// The gRPC exporter connects asynchronously; an unavailable collector cannot
	// delay startup. Export failures are reported once by the exporter wrapper.
	rsc, err := traceResource(ctx, instanceID)
	if err != nil {
		warn(err)
	}
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{Enabled: false}))
	if err != nil {
		warn(err)
		otel.SetTracerProvider(noop.NewTracerProvider())
		return nil
	}
	provider := newTraceProvider(func(err error) {
		liblog.Warnf(ctx, "trace sampler: %v", err)
	},
		sdktrace.WithResource(rsc),
		sdktrace.WithIDGenerator(blocktrace.IDGenerator()),
		sdktrace.WithBatcher(&warningSpanExporter{SpanExporter: exporter, warn: warn}, sdktrace.WithExportTimeout(2*time.Second)),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return provider
}

func newTraceProvider(warn func(error), opts ...sdktrace.TracerProviderOption) *sdktrace.TracerProvider {
	samplerOpts := traceSamplerOptions(warn)
	if len(samplerOpts) == 0 {
		return sdktrace.NewTracerProvider(opts...)
	}
	opts = append(opts, samplerOpts...)
	// v1.45.0 parses the environment before applying WithSampler and reports
	// invalid settings to the global handler even when the sampler is overridden.
	// During startup, suppress that duplicate report while preserving other errors.
	old := otel.GetErrorHandler()
	// The SDK's initial handler delegates to the first replacement. Saving it
	// would make forwarding recurse; preserve its log.Print behavior instead.
	if fmt.Sprintf("%T", old) == "*global.ErrDelegator" {
		old = otel.ErrorHandlerFunc(func(err error) { log.Print(err) })
	}
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		msg := err.Error()
		if strings.HasPrefix(msg, "unsupported sampler:") ||
			strings.HasPrefix(msg, "parsing sampler argument:") ||
			strings.HasPrefix(msg, "invalid trace ID ratio:") {
			return
		}
		old.Handle(err)
	}))
	defer otel.SetErrorHandler(old)
	return sdktrace.NewTracerProvider(opts...)
}

func traceSamplerOptions(warn func(error)) []sdktrace.TracerProviderOption {
	name := strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER")))
	var err error
	fallback := "parentbased_traceidratio"
	switch name {
	case "":
	case "always_on", "always_off", "parentbased_always_on", "parentbased_always_off":
		return nil
	case "traceidratio", "parentbased_traceidratio":
		fallback = name
		arg := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG"))
		ratio, parseErr := strconv.ParseFloat(arg, 64)
		if parseErr == nil && !math.IsNaN(ratio) && !math.IsInf(ratio, 0) && ratio >= 0 && ratio <= 1 {
			return ratioSamplerOptions(name, ratio)
		}
		err = fmt.Errorf("invalid OTEL_TRACES_SAMPLER_ARG %q", arg)
	default:
		err = fmt.Errorf("invalid OTEL_TRACES_SAMPLER %q", name)
	}
	if err != nil {
		warn(fmt.Errorf("%w; using %s(0.05)", err, fallback))
	}
	return ratioSamplerOptions(fallback, 0.05)
}

func ratioSamplerOptions(name string, ratio float64) []sdktrace.TracerProviderOption {
	var sampler sdktrace.Sampler = sdktrace.TraceIDRatioBased(ratio)
	if name == "parentbased_traceidratio" {
		sampler = sdktrace.ParentBased(sampler)
	}
	return []sdktrace.TracerProviderOption{sdktrace.WithSampler(sampler)}
}

type warningSpanExporter struct {
	sdktrace.SpanExporter
	warn    func(error)
	warning sync.Once
}

func (x *warningSpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if err := x.SpanExporter.ExportSpans(ctx, spans); err != nil {
		x.warning.Do(func() { x.warn(err) })
	}
	// Avoid repeated warnings from the batch processor's global error handler.
	return nil
}

func createResource(instanceID string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			// Service metadata
			semconv.ServiceNameKey.String("chuckd"),
			semconv.ServiceVersionKey.String(chuckd.Short()),
			semconv.ServiceInstanceIDKey.String(instanceID),
			// Environment information
			// attribute.String("environment", "production"),
			// attribute.String("region", "us-west-2"),
			// attribute.String("deployment.id", "deployment-123"),
			// Resource limits & utilization context
			// attribute.Int("container.memory.limit", 1024),
			// attribute.Int("container.cpu.limit", 4),
		),
	)
}

func traceResource(ctx context.Context, instanceID string) (*resource.Resource, error) {
	base, baseErr := createResource(instanceID)
	env, envErr := resource.New(ctx, resource.WithFromEnv())
	rsc, mergeErr := resource.Merge(base, env)
	return rsc, errors.Join(baseErr, envErr, mergeErr)
}
