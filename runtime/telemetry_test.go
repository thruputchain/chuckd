package runtime

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thruputchain/chuckd"
	"github.com/thruputchain/chuckd/util/blocktrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func newPprofRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.Host = "127.0.0.1:9101"
	return req
}

func TestTelemetryMuxDisabled(t *testing.T) {
	mux := telemetryMux(PprofConfig{}, false, false)
	for _, name := range []string{"", "cmdline", "profile", "symbol", "trace", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, newPprofRequest(http.MethodGet, "/debug/pprof/"+name))
			if r.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", r.Code)
			}
		})
	}
}

func TestTelemetryMuxEnabled(t *testing.T) {
	mux := telemetryMux(PprofConfig{Enabled: true}, true, false)
	for _, name := range []string{"", "symbol", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, newPprofRequest(http.MethodGet, "/debug/pprof/"+name))
		if r.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", name, r.Code)
		}
		if name == "heap" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(gz)
			gz.Close()
			if err != nil || len(body) == 0 {
				t.Fatalf("invalid heap profile: length=%d, error=%v", len(body), err)
			}
		}
	}
	for _, name := range []string{"profile", "trace"} {
		req := newPprofRequest(http.MethodGet, "/debug/pprof/"+name)
		_, pattern := mux.Handler(req)
		if pattern != "/debug/pprof/"+name {
			t.Fatalf("%s: registered pattern = %q", name, pattern)
		}
	}
}

func TestTelemetryMetrics(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		r := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Host = "evil.example"
		telemetryMux(PprofConfig{Enabled: enabled}, enabled, false).ServeHTTP(r, req)
		if r.Code != http.StatusOK {
			t.Fatalf("enabled=%v: metrics status = %d", enabled, r.Code)
		}
	}
}

func TestTelemetryCmdlineNotFound(t *testing.T) {
	for _, path := range []string{"/debug/pprof/goroutineleak", "/debug/pprof/foo", "/debug/pprof/cmdline", "/debug/pprof/cmdline?debug=1", "/debug/pprof/cmdline/"} {
		r := httptest.NewRecorder()
		telemetryMux(PprofConfig{Enabled: true}, true, false).ServeHTTP(r, newPprofRequest(http.MethodGet, path))
		if r.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want 404", path, r.Code)
		}
	}
}

func TestTelemetryProfileConflicts(t *testing.T) {
	for _, cpu := range []bool{false, true} {
		for _, trace := range []bool{false, true} {
			mux := telemetryMux(PprofConfig{Enabled: true, CPUProfileActive: cpu, TraceProfileActive: trace}, true, false)
			for _, endpoint := range []struct {
				name   string
				flag   string
				active bool
			}{{"profile", "--cpu", cpu}, {"trace", "--trace", trace}} {
				req := newPprofRequest(http.MethodGet, "/debug/pprof/"+endpoint.name)
				handler, _ := mux.Handler(req)
				if !endpoint.active {
					continue
				}
				r := httptest.NewRecorder()
				handler.ServeHTTP(r, req)
				if !strings.Contains(r.Body.String(), endpoint.flag) {
					t.Fatalf("%s: missing explanation: %s", endpoint.name, r.Body.String())
				}
				if r.Code != http.StatusConflict {
					t.Fatalf("cpu=%v trace=%v %s: status = %d, want 409", cpu, trace, endpoint.name, r.Code)
				}
			}
		}
	}
}

func TestPprofIndex(t *testing.T) {
	r := httptest.NewRecorder()
	telemetryMux(PprofConfig{}, true, false).ServeHTTP(r,
		newPprofRequest(http.MethodGet, "/debug/pprof/"))
	if r.Code != http.StatusOK || strings.Contains(r.Body.String(), "cmdline") {
		t.Fatalf("invalid index: %d %s", r.Code, r.Body.String())
	}
	for _, name := range []string{"profile", "trace", "symbol", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
		if !strings.Contains(r.Body.String(), "href=\"/debug/pprof/"+name+"\"") {
			t.Errorf("index missing link for %s", name)
		}
	}
}

func TestPprofSeconds(t *testing.T) {
	mux := telemetryMux(PprofConfig{}, true, false)
	for _, name := range []string{"profile", "trace", "heap", "symbol", ""} {
		for _, seconds := range []string{"61", "1.5", "abc", "-1", "", "9223372036854775808"} {
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, newPprofRequest(http.MethodGet,
				"/debug/pprof/"+name+"?seconds="+seconds))
			if r.Code != http.StatusBadRequest ||
				!strings.Contains(r.Body.String(), "seconds must be an integer between 0 and 60") {
				t.Fatalf("%s seconds=%q: %d %s", name, seconds, r.Code, r.Body.String())
			}
		}
	}
	r := httptest.NewRecorder()
	mux.ServeHTTP(r, newPprofRequest(http.MethodGet, "/debug/pprof/heap?seconds=1"))
	if r.Code != http.StatusOK {
		t.Fatalf("delta heap status = %d: %s", r.Code, r.Body.String())
	}
}

func TestPprofHost(t *testing.T) {
	for _, allowRemote := range []bool{false, true} {
		mux := telemetryMux(PprofConfig{}, true, allowRemote)
		for _, host := range []string{"evil.example", "evil.example:9101", "", "127.0.0.1:9101", "localhost:9101", "LOCALHOST:9101", "[::1]:9101", "127.0.0.2", "[::ffff:127.0.0.2]:9101", "10.0.0.5:9101"} {
			t.Run(fmt.Sprintf("%t/%s", allowRemote, host), func(t *testing.T) {
				want := http.StatusOK
				if host == "" || strings.HasPrefix(host, "evil.example") || (host == "10.0.0.5:9101" && !allowRemote) {
					want = http.StatusForbidden
				}
				req := newPprofRequest(http.MethodGet, "/debug/pprof/heap")
				req.Host = host
				r := httptest.NewRecorder()
				mux.ServeHTTP(r, req)
				if r.Code != want {
					t.Fatalf("status = %d, want %d", r.Code, want)
				}
			})
		}
	}
}

func TestPprofOrigin(t *testing.T) {
	mux := telemetryMux(PprofConfig{}, true, false)
	for _, origin := range []string{"http://evil.example", "http://127.0.0.1:9101", "://invalid", ""} {
		req := newPprofRequest(http.MethodGet, "/debug/pprof/heap")
		req.Header.Set("Origin", origin)
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, req)
		want := http.StatusForbidden
		if origin == "http://127.0.0.1:9101" {
			want = http.StatusOK
		}
		if r.Code != want {
			t.Fatalf("origin %q: status = %d, want %d", origin, r.Code, want)
		}
	}
}

func TestPprofMethodsAndForms(t *testing.T) {
	mux := telemetryMux(PprofConfig{}, true, false)
	for _, tc := range []struct {
		method, target, body, allow, message string
		status                               int
	}{
		{http.MethodPost, "/debug/pprof/heap", "seconds=61", "GET, HEAD", "", 405},
		{http.MethodPut, "/debug/pprof/heap", "", "GET, HEAD", "", 405},
		{http.MethodPut, "/debug/pprof/symbol", "", "GET, HEAD, POST", "", 405},
		{http.MethodPost, "/debug/pprof/symbol", "seconds=61", "", "", 200},
		{http.MethodPost, "/debug/pprof/symbol?seconds=61", "", "", "seconds must be an integer between 0 and 60", 400},
		{http.MethodPost, "/debug/pprof/symbol?seconds=1&seconds=1", "", "", "seconds must be given at most once", 400},
		{http.MethodGet, "/debug/pprof/profile?seconds=1&seconds=61", "", "", "seconds must be given at most once", 400},
		{http.MethodPost, "/debug/pprof/symbol?seconds=1", "seconds=1", "", "", 200},
		{http.MethodGet, "/debug/pprof/heap?seconds=%zz", "", "", "invalid form", 400},
		{http.MethodHead, "/debug/pprof/heap", "", "", "", 200},
		{http.MethodPost, "/debug/pprof/symbol", "", "", "", 200},
		{http.MethodPut, "/debug/pprof/unknown", "", "GET, HEAD", "", 405},
	} {
		req := newPprofRequest(tc.method, tc.target)
		req.Body = io.NopCloser(strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, req)
		if r.Code != tc.status || r.Header().Get("Allow") != tc.allow || !strings.Contains(r.Body.String(), tc.message) {
			t.Fatalf("%s %s body=%q: status=%d Allow=%q response=%q", tc.method, tc.target, tc.body, r.Code, r.Header().Get("Allow"), r.Body.String())
		}
	}
}

func TestPprofSymbolPostBody(t *testing.T) {
	req := newPprofRequest(http.MethodPost, "/debug/pprof/symbol")
	req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf("%#x", reflect.ValueOf(TestPprofIndex).Pointer())))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r := httptest.NewRecorder()
	telemetryMux(PprofConfig{}, true, false).ServeHTTP(r, req)
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "TestPprofIndex") {
		t.Fatalf("symbol response: status=%d body=%q", r.Code, r.Body.String())
	}
}

// net/http/pprof reads r.FormValue("seconds"), which parses a multipart body
// only when r.Form is still nil. limitPprof fills r.Form first, so a multipart
// body cannot smuggle an unchecked seconds value past the cap.
func TestPprofMultipartSecondsIgnored(t *testing.T) {
	var got string
	h := limitPprof(make(chan struct{}, maxConcurrentPprof),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r.FormValue("seconds")
		}))
	body := "--b\r\nContent-Disposition: form-data; name=\"seconds\"\r\n\r\n3600\r\n--b--\r\n"
	req := newPprofRequest(http.MethodGet, "/debug/pprof/profile")
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=b")
	r := httptest.NewRecorder()
	h.ServeHTTP(r, req)
	if r.Code != http.StatusOK || got != "" {
		t.Fatalf("status = %d, handler saw seconds = %q", r.Code, got)
	}
}

func TestPprofConcurrency(t *testing.T) {
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	done := make(chan struct{}, 2)
	h := limitPprof(make(chan struct{}, maxConcurrentPprof),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			entered <- struct{}{}
			<-release
			w.WriteHeader(http.StatusOK)
		}))
	t.Cleanup(func() { close(release) })
	for i := 0; i < 2; i++ {
		go func() {
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
			done <- struct{}{}
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("request did not enter handler")
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if r.Code != http.StatusTooManyRequests {
		t.Fatalf("third request status = %d", r.Code)
	}
	release <- struct{}{}
	release <- struct{}{}
	<-done
	<-done
	go func() { release <- struct{}{} }()
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("request after release status = %d", r.Code)
	}
}

func TestTelemetryReadHeaderTimeout(t *testing.T) {
	srv := newTelemetryServer(http.NewServeMux())
	if got := srv.ReadHeaderTimeout; got != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want 10s", got)
	}
	if got := srv.IdleTimeout; got != 60*time.Second {
		t.Fatalf("IdleTimeout = %v, want 60s", got)
	}
}

func TestTelemetryShutdownDuringProfile(t *testing.T) {
	srv, err := startTelemetry(context.Background(), t.Name(), "127.0.0.1:0", PprofConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		srv.srv.Close()
		srv.mp.Shutdown(context.Background())
	})
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		resp, err := http.Get("http://" + srv.addr.String() + "/debug/pprof/profile?seconds=5")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()
	time.Sleep(200 * time.Millisecond)
	select {
	case <-requestDone:
		t.Fatal("profile request finished before shutdown")
	default:
	}
	done := make(chan error, 1)
	go func() { done <- srv.shutdown(context.Background()) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not return within 1s")
	}
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("profile client did not disconnect after shutdown")
	}
}

func TestLoopback(t *testing.T) {
	for _, tc := range []struct {
		ip   string
		want bool
	}{{"127.0.0.1", true}, {"::1", true}, {"::ffff:127.0.0.1", true},
		{"0.0.0.0", false}, {"::", false}, {"192.0.2.1", false}} {
		if got := isLoopback(net.ParseIP(tc.ip)); got != tc.want {
			t.Errorf("isLoopback(%s) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}

func cleanupTelemetry(t *testing.T, srv *telemetry) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.shutdown(ctx); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
}

func TestStartTelemetry(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:0", "[::1]:0", ":0", "0.0.0.0:0"} {
		for _, opt := range []struct {
			name string
			flag bool
			env  string
		}{{"default", false, ""}, {"flag", true, ""},
			{"env", false, "1"}, {"env true", false, "true"}} {
			t.Run(addr+"/"+opt.name, func(t *testing.T) {
				t.Setenv("GROUNDHOG_PPROF_ALLOW_REMOTE", opt.env)
				srv, err := startTelemetry(context.Background(), t.Name(), addr,
					PprofConfig{Enabled: true, AllowRemote: opt.flag})
				if err != nil {
					if addr == "[::1]:0" {
						t.Skipf("IPv6 loopback unavailable: %v", err)
					}
					t.Fatal(err)
				}
				cleanupTelemetry(t, srv)
				tcp := srv.addr.(*net.TCPAddr)
				host := tcp.IP.String()
				if tcp.IP.IsUnspecified() {
					host = "127.0.0.1"
				}
				base := "http://" + net.JoinHostPort(host, fmt.Sprint(tcp.Port))
				want := http.StatusNotFound
				if isLoopback(tcp.IP) || opt.flag || opt.env == "1" {
					want = http.StatusOK
				}
				for _, endpoint := range []struct {
					path   string
					status int
				}{
					{"/metrics", http.StatusOK}, {"/debug/pprof/heap", want},
				} {
					resp, err := http.Get(base + endpoint.path)
					if err != nil {
						t.Fatal(err)
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					if resp.StatusCode != endpoint.status {
						t.Errorf("%s status = %d, want %d", endpoint.path, resp.StatusCode, endpoint.status)
					}
				}
			})
		}
	}
}

func TestSetupTelemetryBindError(t *testing.T) {
	t.Setenv("GROUNDHOG_PPROF_ALLOW_REMOTE", "")
	srv, err := startTelemetry(context.Background(), t.Name(), "127.0.0.1:0", PprofConfig{})
	if err != nil {
		t.Fatal(err)
	}
	cleanupTelemetry(t, srv)
	shutdown, err := SetupTelemetry(context.Background(), t.Name(), srv.addr.String(), PprofConfig{})
	if err == nil {
		t.Cleanup(func() { shutdown(context.Background()) })
		t.Fatal("second listener did not return a bind error")
	}
}

func TestTelemetryResources(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("OTEL_SERVICE_NAME", "")
	for _, build := range []func(string) (*resource.Resource, error){
		createResource,
		func(id string) (*resource.Resource, error) { return traceResource(context.Background(), id) },
	} {
		rsc, err := build("x")
		if err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{"service.name": "chuckd", "service.version": chuckd.Short(), "service.instance.id": "x"} {
			got, ok := rsc.Set().Value(attribute.Key(key))
			if !ok || got.AsString() != want {
				t.Errorf("%s = %v, want %q", key, got, want)
			}
		}
	}
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.version=override,service.instance.id=env")
	t.Setenv("OTEL_SERVICE_NAME", "env-service")
	rsc, err := traceResource(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"service.name": "env-service", "service.version": "override", "service.instance.id": "env"} {
		got, _ := rsc.Set().Value(attribute.Key(key))
		if got.AsString() != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
}

type failingSpanExporter struct{}

func (failingSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return errors.New("export failed")
}

func (failingSpanExporter) Shutdown(context.Context) error { return nil }

func TestExportWarningAfterResourceWarning(t *testing.T) {
	warnings := 0
	warn := func(error) { warnings++ }
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "invalid")
	_, err := traceResource(context.Background(), "x")
	if err == nil {
		t.Fatal("expected resource error")
	}
	warn(err)
	exporter := &warningSpanExporter{SpanExporter: failingSpanExporter{}, warn: warn}
	for range 2 {
		if err := exporter.ExportSpans(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if warnings != 2 {
		t.Fatalf("warnings = %d, want resource warning and one export warning", warnings)
	}
}

func TestTracingBlackHoleShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		var connections []net.Conn
		defer func() {
			for _, conn := range connections {
				conn.Close()
			}
		}()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			connections = append(connections, conn)
		}
	}()
	t.Cleanup(func() { listener.Close(); <-done })
	oldProvider, oldHandler, oldPropagator := otel.GetTracerProvider(), otel.GetErrorHandler(), otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(oldProvider)
		otel.SetErrorHandler(oldHandler)
		otel.SetTextMapPropagator(oldPropagator)
	})
	t.Setenv("OTEL_SDK_DISABLED", "false")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://"+listener.Addr().String())
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	t.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	provider := setupTracing(context.Background(), "test")
	if provider == nil {
		t.Fatal("missing tracer provider")
	}
	shutdown := newShutdown(provider, nil)
	t.Cleanup(func() { shutdown(context.Background()) })
	id := blocktrace.TraceID([32]byte{1}, 42)
	ctx := blocktrace.WithTraceID(context.Background(), id)
	for range 3 {
		_, span := provider.Tracer("test").Start(ctx, "black-hole")
		if span.SpanContext().TraceID() != id {
			t.Error("provider did not use block trace ID")
		}
		span.End()
	}
	start := time.Now()
	shutdown(context.Background())
	if elapsed := time.Since(start); elapsed > 7*time.Second {
		t.Fatalf("shutdown took %v", elapsed)
	}
}

func TestTracingEndpointSwitch(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "false")
	t.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	oldProvider, oldHandler, oldPropagator := otel.GetTracerProvider(), otel.GetErrorHandler(), otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(oldProvider)
		otel.SetErrorHandler(oldHandler)
		otel.SetTextMapPropagator(oldPropagator)
	})
	for _, endpoint := range []string{"", "http://127.0.0.1:4317"} {
		t.Run(endpoint, func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", endpoint)
			ctx := context.Background()
			provider := setupTracing(ctx, "test")
			_, span := otel.Tracer("test").Start(ctx, "switch")
			if span.IsRecording() != (endpoint != "") {
				t.Fatalf("recording = %v for endpoint %q", span.IsRecording(), endpoint)
			}
			// Do not queue a span for the unreachable test endpoint.
			if provider != nil {
				if err := provider.Shutdown(ctx); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

type fixedTraceID byte

func (b fixedTraceID) NewIDs(context.Context) (trace.TraceID, trace.SpanID) {
	var id trace.TraceID
	for i := range id {
		id[i] = byte(b)
	}
	return id, trace.SpanID{1}
}

func (b fixedTraceID) NewSpanID(context.Context, trace.TraceID) trace.SpanID {
	return trace.SpanID{1}
}

func TestTraceSamplerEnvironment(t *testing.T) {
	for _, tc := range []struct {
		name string
		low  bool
		high bool
	}{
		{"always_on", true, true},
		{"always_off", false, false},
		{"traceidratio", true, false},
		{"parentbased_always_on", true, true},
		{"parentbased_always_off", false, false},
		{"parentbased_traceidratio", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_SAMPLER", tc.name)
			t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.5")
			opts := traceSamplerOptions(func(err error) { t.Fatal(err) })
			for _, sample := range []struct {
				id   fixedTraceID
				want bool
			}{{0x40, tc.low}, {0xc0, tc.high}} {
				assertTraceSample(t, opts, sample.id, context.Background(), sample.want)
				for _, remote := range []bool{false, true} {
					for _, sampled := range []bool{false, true} {
						flags := trace.TraceFlags(0)
						if sampled {
							flags = trace.FlagsSampled
						}
						id, sid := sample.id.NewIDs(context.Background())
						parent := trace.NewSpanContext(trace.SpanContextConfig{
							TraceID: id, SpanID: sid, TraceFlags: flags, Remote: remote,
						})
						ctx := trace.ContextWithSpanContext(context.Background(), parent)
						want := sample.want
						if tc.name == "parentbased_always_on" ||
							tc.name == "parentbased_always_off" ||
							tc.name == "parentbased_traceidratio" {
							want = sampled
						}
						assertTraceSample(t, opts, sample.id, ctx, want)
					}
				}
			}
		})
	}
}

func assertTraceSample(t *testing.T, opts []sdktrace.TracerProviderOption,
	id fixedTraceID, ctx context.Context, want bool,
) {
	t.Helper()
	all := append([]sdktrace.TracerProviderOption{sdktrace.WithIDGenerator(id)}, opts...)
	p := newTraceProvider(func(error) {}, all...)
	t.Cleanup(func() {
		if err := p.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})
	_, span := p.Tracer("test").Start(ctx, "sample")
	defer span.End()
	if span.IsRecording() != want {
		t.Errorf("sampler=%q id=%x parent=%v: recording=%v, want %v",
			os.Getenv("OTEL_TRACES_SAMPLER"), byte(id), trace.SpanContextFromContext(ctx),
			span.IsRecording(), want)
	}
}

func TestTraceSamplerDefaultAndFallback(t *testing.T) {
	old := otel.GetErrorHandler()
	t.Cleanup(func() { otel.SetErrorHandler(old) })
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		t.Errorf("unexpected SDK error: %v", err)
	}))
	t.Run("unset", func(t *testing.T) {
		t.Setenv("OTEL_TRACES_SAMPLER", "")
		if err := os.Unsetenv("OTEL_TRACES_SAMPLER"); err != nil {
			t.Fatal(err)
		}
		assertTraceSample(t, nil, 0x0a, context.Background(), true)
		assertTraceSample(t, nil, 0x0f, context.Background(), false)
	})
	for _, tc := range []struct {
		name string
		arg  string
		warn int
	}{
		{"", "", 0},
		{"", "1", 0},
		{"invalid", "0.5", 1},
		{"traceidratio", "", 1},
		{"parentbased_traceidratio", "invalid", 1},
		{"traceidratio", "NaN", 1},
		{"traceidratio", "Inf", 1},
		{"traceidratio", "-0.1", 1},
		{"traceidratio", "1.1", 1},
	} {
		t.Run(tc.name+"/"+tc.arg, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_SAMPLER", tc.name)
			t.Setenv("OTEL_TRACES_SAMPLER_ARG", tc.arg)
			warnings := 0
			p := newTraceProvider(func(error) { warnings++ })
			if err := p.Shutdown(context.Background()); err != nil {
				t.Fatal(err)
			}
			if warnings != tc.warn {
				t.Fatalf("warnings=%d, want %d", warnings, tc.warn)
			}
			assertTraceSample(t, nil, 0x0a, context.Background(), true)
			assertTraceSample(t, nil, 0x0f, context.Background(), false)
		})
	}
}

func TestTracingSDKDisabled(t *testing.T) {
	old := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(old) })
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4317")
	t.Setenv("OTEL_TRACES_SAMPLER", "invalid")
	for _, disabled := range []string{"true", "TRUE"} {
		t.Run(disabled, func(t *testing.T) {
			t.Setenv("OTEL_SDK_DISABLED", disabled)
			p := setupTracing(context.Background(), "test")
			if p != nil {
				t.Fatal("disabled tracing created an SDK provider")
			}
			if _, ok := otel.GetTracerProvider().(noop.TracerProvider); !ok {
				t.Fatalf("disabled provider type: %T", otel.GetTracerProvider())
			}
			_, span := otel.Tracer("test").Start(context.Background(), "disabled")
			defer span.End()
			if span.IsRecording() {
				t.Fatal("disabled tracing recorded a span")
			}
		})
	}
}

func TestRatioSamplerArguments(t *testing.T) {
	old := otel.GetErrorHandler()
	t.Cleanup(func() { otel.SetErrorHandler(old) })
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		t.Errorf("unexpected SDK error: %v", err)
	}))
	for _, name := range []string{"traceidratio", "parentbased_traceidratio"} {
		for _, tc := range []struct {
			arg      string
			missing  bool
			warn     int
			low      fixedTraceID
			high     fixedTraceID
			lowWant  bool
			highWant bool
		}{
			{arg: "0.5", low: 0x7f, high: 0x81, lowWant: true},
			{arg: " 0.25 ", low: 0x3f, high: 0x41, lowWant: true},
			{arg: "0", low: 0x01, high: 0xff},
			{arg: "1", low: 0x01, high: 0xff, lowWant: true, highWant: true},
			{missing: true, warn: 1, low: 0x0a, high: 0x0f, lowWant: true},
			{arg: "", warn: 1, low: 0x0a, high: 0x0f, lowWant: true},
			{arg: "abc", warn: 1, low: 0x0a, high: 0x0f, lowWant: true},
			{arg: "NaN", warn: 1, low: 0x0a, high: 0x0f, lowWant: true},
			{arg: "Inf", warn: 1, low: 0x0a, high: 0x0f, lowWant: true},
			{arg: "-Inf", warn: 1, low: 0x0a, high: 0x0f, lowWant: true},
			{arg: "-0.1", warn: 1, low: 0x0a, high: 0x0f, lowWant: true},
			{arg: "1.1", warn: 1, low: 0x0a, high: 0x0f, lowWant: true},
		} {
			label := tc.arg
			if tc.missing {
				label = "missing"
			}
			t.Run(name+"/"+label, func(t *testing.T) {
				t.Setenv("OTEL_TRACES_SAMPLER", " "+strings.ToUpper(name)+" ")
				t.Setenv("OTEL_TRACES_SAMPLER_ARG", tc.arg)
				if tc.missing {
					if err := os.Unsetenv("OTEL_TRACES_SAMPLER_ARG"); err != nil {
						t.Fatal(err)
					}
				}
				for _, sample := range []struct {
					id   fixedTraceID
					want bool
				}{{tc.low, tc.lowWant}, {tc.high, tc.highWant}} {
					warnings := 0
					p := newTraceProvider(func(err error) {
						warnings++
						if !strings.Contains(err.Error(), "using "+name+"(0.05)") {
							t.Errorf("incorrect fallback warning: %v", err)
						}
					}, sdktrace.WithIDGenerator(sample.id))
					t.Cleanup(func() {
						if err := p.Shutdown(context.Background()); err != nil {
							t.Error(err)
						}
					})
					if warnings != tc.warn {
						t.Fatalf("warnings=%d, want %d", warnings, tc.warn)
					}
					check := func(ctx context.Context, want bool) {
						t.Helper()
						_, span := p.Tracer("test").Start(ctx, "sample")
						defer span.End()
						if span.SpanContext().IsSampled() != want {
							t.Errorf("id=%x parent=%v: sampled=%v, want %v", byte(sample.id),
								trace.SpanContextFromContext(ctx), span.SpanContext().IsSampled(), want)
						}
					}
					check(context.Background(), sample.want)
					for _, sampled := range []bool{false, true} {
						flags := trace.TraceFlags(0)
						if sampled {
							flags = trace.FlagsSampled
						}
						id, sid := sample.id.NewIDs(context.Background())
						parent := trace.NewSpanContext(trace.SpanContextConfig{
							TraceID: id, SpanID: sid, TraceFlags: flags,
						})
						want := sample.want
						if name == "parentbased_traceidratio" {
							want = sampled
						}
						check(trace.ContextWithSpanContext(context.Background(), parent), want)
					}
				}
			})
		}
	}
}

func TestBlockTraceSamplerCrossProcessBoundary(t *testing.T) {
	old := otel.GetErrorHandler()
	t.Cleanup(func() { otel.SetErrorHandler(old) })
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		t.Errorf("unexpected SDK error: %v", err)
	}))
	var network [32]byte
	for i := range network {
		network[i] = 0x11
	}
	id := blocktrace.TraceID(network, 42)
	if id.String() != "553a3192399806a83a6f64004c4cab48" {
		t.Fatalf("unexpected shared trace ID: %s", id)
	}
	// The decision is uint64(lowBE64(traceID) >> 1) < uint64(ratio * 2^63),
	// the same rule as opentelemetry_sdk 0.33 sample_based_on_probability used
	// by scpd. This vector's threshold is 0x3a6f64004c4cab48 / 2^64 ≈ 0.2283.
	for _, name := range []string{"traceidratio", "parentbased_traceidratio"} {
		for _, tc := range []struct {
			arg  string
			want bool
		}{{"0.22", false}, {"0.23", true}} {
			t.Run(name+"/"+tc.arg, func(t *testing.T) {
				t.Setenv("OTEL_TRACES_SAMPLER", name)
				t.Setenv("OTEL_TRACES_SAMPLER_ARG", tc.arg)
				p := newTraceProvider(func(err error) { t.Errorf("unexpected warning: %v", err) },
					sdktrace.WithIDGenerator(blocktrace.IDGenerator()))
				t.Cleanup(func() {
					if err := p.Shutdown(context.Background()); err != nil {
						t.Error(err)
					}
				})
				_, span := p.Tracer("test").Start(blocktrace.WithTraceID(context.Background(), id), "block")
				defer span.End()
				if span.SpanContext().TraceID() != id || span.SpanContext().IsSampled() != tc.want {
					t.Fatalf("trace ID=%s sampled=%v, want %s sampled=%v", span.SpanContext().TraceID(),
						span.SpanContext().IsSampled(), id, tc.want)
				}
			})
		}
	}
}
