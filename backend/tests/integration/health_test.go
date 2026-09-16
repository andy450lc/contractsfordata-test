package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	cerrors "github.com/cockroachdb/errors"
	"github.com/labstack/echo/v5"
	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/boot"
)

type readyzResponse struct {
	Status string `json:"status"`
	Checks []struct {
		Name    string `json:"name"`
		Healthy bool   `json:"healthy"`
		Error   string `json:"error"`
	} `json:"checks"`
}

// logLine is the JSON shape every emitted log line decodes into.
type logLine struct {
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	TraceID   string `json:"trace_id"`
	SpanID    string `json:"span_id"`
	URI       string `json:"uri"`
	Status    int    `json:"status"`
}

// failingRouteOption registers a route whose handler fails the way a
// store-level external call fails: a stack captured at the origin,
// wrapped upward with operation context.
func failingRouteOption() fx.Option {
	return fx.Invoke(func(s *boot.Server) {
		s.Echo.GET("/v1/test/fail", func(c *echo.Context) error {
			origin := cerrors.New("vendor api exploded")
			return fmt.Errorf("charging flux capacitor: %w", origin)
		})
	})
}

// Livez answers success without consulting any dependency.
func TestLivezReturnsStaticOK(t *testing.T) {
	app := startApp(t, nil)

	resp, body := get(t, app.BaseURL+"/livez")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("livez status = %d, want 200 (body: %s)", resp.StatusCode, body)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("livez body not JSON: %v (body: %s)", err, body)
	}
	if payload["status"] != "ok" {
		t.Fatalf(`livez status field = %q, want "ok"`, payload["status"])
	}
}

// Readyz verifies the database and answers fast.
func TestReadyzHealthy(t *testing.T) {
	app := startApp(t, nil)

	start := time.Now()
	resp, body := get(t, app.BaseURL+"/readyz")
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("readyz status = %d, want 200 (body: %s)", resp.StatusCode, body)
	}
	if elapsed > time.Second {
		t.Fatalf("readyz took %v, want < 1s", elapsed)
	}

	var payload readyzResponse
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("readyz body not JSON: %v (body: %s)", err, body)
	}
	if payload.Status != "ok" {
		t.Fatalf(`readyz status = %q, want "ok"`, payload.Status)
	}
	if len(payload.Checks) != 1 || payload.Checks[0].Name != "database" || !payload.Checks[0].Healthy {
		t.Fatalf("readyz checks = %+v, want single healthy database check", payload.Checks)
	}
}

// Losing the database degrades readiness to 503 while liveness stays
// healthy. Recovery needs no restart.
func TestReadyzDegradesAndRecoversWithDatabaseLoss(t *testing.T) {
	app := startApp(t, nil)

	resume := pausePostgres(t)

	resp, body := get(t, app.BaseURL+"/readyz")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("readyz with DB down = %d, want 503 (body: %s)", resp.StatusCode, body)
	}

	var payload readyzResponse
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("readyz degraded body not JSON: %v (body: %s)", err, body)
	}
	if payload.Status != "unavailable" {
		t.Fatalf(`degraded readyz status = %q, want "unavailable"`, payload.Status)
	}
	if len(payload.Checks) != 1 || payload.Checks[0].Healthy {
		t.Fatalf("degraded readyz checks = %+v, want unhealthy database check", payload.Checks)
	}
	if payload.Checks[0].Error == "" {
		t.Fatal("degraded database check should carry an operational error message")
	}
	if strings.Contains(payload.Checks[0].Error, "sow:sow") {
		t.Fatal("check error leaks connection credentials")
	}

	livezResp, _ := get(t, app.BaseURL+"/livez")
	if livezResp.StatusCode != http.StatusOK {
		t.Fatalf("livez with DB down = %d, want 200", livezResp.StatusCode)
	}

	resume()

	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, _ := get(t, app.BaseURL+"/readyz")
		if resp.StatusCode == http.StatusOK {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("readyz did not recover after database restart")
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// Every response carries the mandated security headers and a request id.
// An incoming request id is honored.
func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	app := startApp(t, nil)

	for _, path := range []string{"/livez", "/v1/unknown"} {
		resp, _ := get(t, app.BaseURL+path)

		want := map[string]string{
			"X-Content-Type-Options":  "nosniff",
			"X-Frame-Options":         "DENY",
			"Content-Security-Policy": "default-src 'none'",
			"Referrer-Policy":         "no-referrer",
		}
		for header, value := range want {
			if got := resp.Header.Get(header); got != value {
				t.Fatalf("%s: header %s = %q, want %q", path, header, got, value)
			}
		}
		if resp.Header.Get("X-Request-Id") == "" {
			t.Fatalf("%s: missing X-Request-ID on response", path)
		}
	}

	// HSTS is emitted only for TLS-terminated traffic. The forwarded-proto
	// header simulates the deployed proxy.
	tlsReq, _ := http.NewRequest(http.MethodGet, app.BaseURL+"/livez", nil)
	tlsReq.Header.Set("X-Forwarded-Proto", "https")
	tlsResp, err := http.DefaultClient.Do(tlsReq) //nolint:forbidigo // test hits our own server
	if err != nil {
		t.Fatalf("GET with forwarded proto: %v", err)
	}
	defer tlsResp.Body.Close()
	if got := tlsResp.Header.Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=31536000") {
		t.Fatalf("HSTS missing on https-forwarded response: %q", got)
	}

	req, _ := http.NewRequest(http.MethodGet, app.BaseURL+"/livez", nil)
	req.Header.Set("X-Request-ID", "caller-supplied-id")
	resp, err := http.DefaultClient.Do(req) //nolint:forbidigo // test hits our own server
	if err != nil {
		t.Fatalf("GET with request id: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("X-Request-Id"); got != "caller-supplied-id" {
		t.Fatalf("incoming X-Request-ID not honored: got %q", got)
	}
}

// Exceeding the per-client rate answers 429. The 429s appear in the
// request metrics. Probes stay exempt.
func TestRateLimitTripsAndIsMetered(t *testing.T) {
	app := startApp(t, map[string]string{
		"RATE_LIMIT_RPS":   "2",
		"RATE_LIMIT_BURST": "3",
	})

	saw429 := false
	for range 25 {
		resp, body := get(t, app.BaseURL+"/v1/limited")
		if resp.StatusCode == http.StatusTooManyRequests {
			saw429 = true
			if !strings.Contains(body, `"error"`) {
				t.Fatalf("429 body missing standard envelope: %s", body)
			}
		}
	}
	if !saw429 {
		t.Fatal("burst of 25 requests never tripped the rate limit (rps=2, burst=3)")
	}

	for range 10 {
		resp, _ := get(t, app.BaseURL+"/livez")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("livez rate limited: %d", resp.StatusCode)
		}
	}

	_, metrics := get(t, app.BaseURL+"/metrics")
	if !strings.Contains(metrics, `status="429"`) {
		t.Fatal("429 responses not visible in request metrics")
	}
}

// An unexpected failure returns a generic envelope. Exactly one error
// log line carries the wrapped operation chain with its correlation ids.
func TestUnexpectedErrorLoggedExactlyOnce(t *testing.T) {
	app := startApp(t, nil, failingRouteOption())

	resp, body := get(t, app.BaseURL+"/v1/test/fail")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("failing route status = %d, want 500", resp.StatusCode)
	}
	if strings.Contains(body, "flux capacitor") {
		t.Fatalf("internal error detail leaked to client: %s", body)
	}

	waitForAccessLog(t, app, "/v1/test/fail")

	// The central handler is the only place error content is logged.
	var errLines []string
	for _, raw := range app.Logs.Lines() {
		if strings.Contains(raw, "charging flux capacitor") {
			errLines = append(errLines, raw)
		}
	}
	if len(errLines) != 1 {
		t.Fatalf("error content appeared in %d log lines, want exactly 1; logs:\n%s", len(errLines), app.Logs.String())
	}

	var errLine logLine
	if err := json.Unmarshal([]byte(errLines[0]), &errLine); err != nil {
		t.Fatalf("error line not JSON: %v", err)
	}
	if errLine.Level != "ERROR" {
		t.Fatalf("central error log level = %q, want ERROR", errLine.Level)
	}
	if errLine.RequestID == "" {
		t.Fatalf("error log missing request_id: %s", errLines[0])
	}
	if errLine.TraceID == "" || errLine.SpanID == "" {
		t.Fatalf("error log missing trace correlation: %s", errLines[0])
	}
	// The stack captured at the origin locates the failing frame.
	if !strings.Contains(errLines[0], "vendor api exploded") {
		t.Fatalf("error log missing origin error: %s", errLines[0])
	}
}

func parseLines(t *testing.T, raw []string) []logLine {
	t.Helper()

	lines := make([]logLine, 0, len(raw))
	for _, l := range raw {
		var parsed logLine
		if err := json.Unmarshal([]byte(l), &parsed); err != nil {
			t.Fatalf("log line is not JSON: %v (line: %s)", err, l)
		}
		lines = append(lines, parsed)
	}
	return lines
}

// waitForAccessLog polls until an access-log line for uri appears. The
// request logger writes after the response is sent.
func waitForAccessLog(t *testing.T, app *testApp, uri string) []logLine {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		lines := parseLines(t, app.Logs.Lines())
		for _, l := range lines {
			if l.URI == uri {
				return lines
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no access log for %s appeared; logs:\n%s", uri, app.Logs.String())
		}
		time.Sleep(50 * time.Millisecond)
	}
}
