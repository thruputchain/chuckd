package runtime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/groundhogchain/chuckd"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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
}

func (t *telemetry) shutdown(ctx context.Context) error {
	return errors.Join(t.srv.Close(), t.mp.Shutdown(ctx))
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
	return &telemetry{addr: ln.Addr(), srv: srv, mp: meterProvider}, nil
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

func createResource(instanceID string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
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
