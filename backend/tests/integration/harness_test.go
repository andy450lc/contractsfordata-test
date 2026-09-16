// Package integration boots the real fx application against a real
// Postgres and exercises it over HTTP. Setup applies the goose migrations.
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/moby/moby/client"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/boot"
	"github.com/pixels-two/sow/backend/internal/logging"
	"github.com/pixels-two/sow/backend/internal/migrations"
	"github.com/pixels-two/sow/backend/internal/stores"
)

var (
	testDSN     string
	pgContainer *tcpostgres.PostgresContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("sow"),
		tcpostgres.WithUsername("sow"),
		tcpostgres.WithPassword("sow"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("starting postgres container: %v", err)
	}
	pgContainer = container

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("resolving container DSN: %v", err)
	}
	testDSN = dsn

	if err := applyMigrations(dsn); err != nil {
		log.Fatalf("applying migrations: %v", err)
	}

	code := m.Run()

	_ = testcontainers.TerminateContainer(container)
	os.Exit(code)
}

// applyMigrations runs goose up from the embedded migration FS — the same
// files the goose CLI applies.
func applyMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("running goose up: %w", err)
	}
	return nil
}

// syncBuffer is a concurrency-safe log sink for asserting on emitted lines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// WaitForLine waits for a captured log line containing want and returns
// every line captured by then. The test fails when no such line arrives
// within two seconds.
func (b *syncBuffer) WaitForLine(t *testing.T, want string) []string {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		lines := b.Lines()
		for _, line := range lines {
			if strings.Contains(line, want) {
				return lines
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no log line contains %q", want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Lines returns non-empty log lines captured so far.
func (b *syncBuffer) Lines() []string {
	var lines []string
	for _, l := range strings.Split(b.String(), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

type testApp struct {
	App     *fx.App
	Server  *boot.Server
	BaseURL string
	Logs    *syncBuffer
}

// testSessionCookieKey is a fixed 32-byte AES key for the suite. Real
// deployments generate their own.
const testSessionCookieKey = "k1:A9Keki18RmRaac8WO5FQeo9czkzVdL2hFHzp/aDm4IQ="

// defaultEnv is the baseline test configuration. Per-test overrides merge
// on top. Rate limits are high so unrelated tests never trip them.
func defaultEnv() map[string]string {
	return map[string]string{
		"STAGE":                       "dev",
		"PORT":                        "0",
		"DATABASE_URL":                testDSN,
		"DB_POOL_MAX_CONNS":           "10",
		"DB_POOL_MIN_CONNS":           "2",
		"CORS_ALLOWED_ORIGINS":        "http://allowed.example",
		"RATE_LIMIT_RPS":              "5000",
		"RATE_LIMIT_BURST":            "5000",
		"AUTH_RATE_LIMIT_RPS":         "5000",
		"AUTH_RATE_LIMIT_BURST":       "5000",
		"SHUTDOWN_GRACE_SECONDS":      "5",
		"WORKOS_API_KEY":              "sk_test",
		"WORKOS_CLIENT_ID":            "client_test",
		"WORKOS_WEBHOOK_SECRET":       "whsec_test",
		"WORKOS_REDIRECT_URI":         "http://localhost:8080/v1/auth/callback",
		"WORKOS_JWKS_URL":             "",
		"WORKOS_BASE_URL":             "",
		"WEB_APP_URL":                 "http://localhost:5173",
		"SESSION_COOKIE_KEY":          testSessionCookieKey,
		"SESSION_MAX_AGE_SECONDS":     "604800",
		"SENTRY_DSN":                  "",
		"OTEL_EXPORTER_OTLP_ENDPOINT": "",
	}
}

func applyEnv(t *testing.T, overrides map[string]string) {
	t.Helper()

	env := defaultEnv()
	for k, v := range overrides {
		env[k] = v
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

// buildApp assembles the production fx graph plus test instrumentation
// (captured logs, populated server handle). It does not start the app.
func buildApp(t *testing.T, overrides map[string]string, extra ...fx.Option) (*fx.App, **boot.Server, *syncBuffer) {
	t.Helper()

	applyEnv(t, overrides)

	logs := &syncBuffer{}
	var srv *boot.Server

	opts := []fx.Option{
		boot.AppOptions(),
		fx.Decorate(func() *slog.Logger {
			return logging.NewWithWriter("dev", logs)
		}),
		fx.Populate(&srv),
	}
	opts = append(opts, extra...)

	app := fx.New(opts...)
	return app, &srv, logs
}

// startApp boots the real application on a random port and returns a
// handle with its base URL and captured logs. Cleanup stops the app.
// Extra fx options let a test register test-only routes before start.
func startApp(t *testing.T, overrides map[string]string, extra ...fx.Option) *testApp {
	t.Helper()

	app, srv, logs := buildApp(t, overrides, extra...)

	startCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		t.Fatalf("starting app: %v", err)
	}

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = app.Stop(stopCtx)
	})

	addr := (*srv).Addr()
	if addr == "" {
		t.Fatal("server address empty after start")
	}
	_, port, ok := strings.Cut(addr, ":")
	for ok && strings.Contains(port, ":") { // [::]:port form
		_, port, ok = strings.Cut(port, ":")
	}

	return &testApp{
		App:     app,
		Server:  *srv,
		BaseURL: "http://127.0.0.1:" + port,
		Logs:    logs,
	}
}

// get issues a GET and returns response + fully-read body.
func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()

	resp, err := http.Get(url) //nolint:forbidigo // test helper hits our own server
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body of %s: %v", url, err)
	}
	return resp, string(body)
}

// pausePostgres freezes the database container and returns the resume
// function. The mapped port survives, so callers exercise the bounded
// check timeouts against a frozen socket. Resuming twice is safe.
func pausePostgres(t *testing.T) func() {
	t.Helper()
	ctx := context.Background()

	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		t.Fatalf("creating docker provider: %v", err)
	}
	t.Cleanup(func() { _ = provider.Close() })

	if _, err := provider.Client().ContainerPause(ctx, pgContainer.GetContainerID(), client.ContainerPauseOptions{}); err != nil {
		t.Fatalf("pausing postgres container: %v", err)
	}

	resumed := false
	resume := func() {
		if resumed {
			return
		}
		resumed = true
		if _, err := provider.Client().ContainerUnpause(ctx, pgContainer.GetContainerID(), client.ContainerUnpauseOptions{}); err != nil {
			t.Fatalf("unpausing postgres container: %v", err)
		}
	}
	t.Cleanup(resume)
	return resume
}

// newUserStore opens a pool on the test database for direct row checks.
func newUserStore(t *testing.T) *stores.UserStore {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), testDSN)
	if err != nil {
		t.Fatalf("opening pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return stores.NewUserStore(pool)
}
