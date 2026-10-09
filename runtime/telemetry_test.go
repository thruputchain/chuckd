package runtime

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTelemetryMuxDisabled(t *testing.T) {
	mux := telemetryMux(PprofConfig{}, false)
	for _, name := range []string{"", "cmdline", "profile", "symbol", "trace", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/debug/pprof/"+name, nil))
			if r.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", r.Code)
			}
		})
	}
}

func TestTelemetryMuxEnabled(t *testing.T) {
	mux := telemetryMux(PprofConfig{Enabled: true}, true)
	for _, name := range []string{"", "symbol", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/debug/pprof/"+name, nil))
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
		req := httptest.NewRequest(http.MethodGet, "/debug/pprof/"+name, nil)
		_, pattern := mux.Handler(req)
		if pattern != "/debug/pprof/"+name {
			t.Fatalf("%s: registered pattern = %q", name, pattern)
		}
	}
}

func TestTelemetryMetrics(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		r := httptest.NewRecorder()
		telemetryMux(PprofConfig{Enabled: enabled}, enabled).ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if r.Code != http.StatusOK {
			t.Fatalf("enabled=%v: metrics status = %d", enabled, r.Code)
		}
	}
}

func TestTelemetryCmdlineNotFound(t *testing.T) {
	for _, path := range []string{"/debug/pprof/goroutineleak", "/debug/pprof/foo", "/debug/pprof/cmdline", "/debug/pprof/cmdline?debug=1", "/debug/pprof/cmdline/"} {
		r := httptest.NewRecorder()
		telemetryMux(PprofConfig{Enabled: true}, true).ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want 404", path, r.Code)
		}
	}
}

func TestTelemetryProfileConflicts(t *testing.T) {
	for _, cpu := range []bool{false, true} {
		for _, trace := range []bool{false, true} {
			mux := telemetryMux(PprofConfig{Enabled: true, CPUProfileActive: cpu, TraceProfileActive: trace}, true)
			for _, endpoint := range []struct {
				name   string
				flag   string
				active bool
			}{{"profile", "--cpu", cpu}, {"trace", "--trace", trace}} {
				req := httptest.NewRequest(http.MethodGet, "/debug/pprof/"+endpoint.name, nil)
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
	telemetryMux(PprofConfig{}, true).ServeHTTP(r,
		httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
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
	mux := telemetryMux(PprofConfig{}, true)
	for _, name := range []string{"profile", "trace", "heap", "symbol", ""} {
		for _, seconds := range []string{"61", "abc", "-1", ""} {
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet,
				"/debug/pprof/"+name+"?seconds="+seconds, nil))
			if r.Code != http.StatusBadRequest ||
				!strings.Contains(r.Body.String(), "seconds must be an integer between 0 and 60") {
				t.Fatalf("%s seconds=%q: %d %s", name, seconds, r.Code, r.Body.String())
			}
		}
	}
	r := httptest.NewRecorder()
	mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/debug/pprof/heap?seconds=1", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("delta heap status = %d: %s", r.Code, r.Body.String())
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
	if got := newTelemetryServer(http.NewServeMux()).ReadHeaderTimeout; got != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want 10s", got)
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
