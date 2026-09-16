package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pixels-two/sow/backend/internal/mail"
)

// fakeMailer captures every message handed to it. A test asserts on the
// recipient, subject, and body a service produced. No mail leaves the
// process.
type fakeMailer struct {
	mu      sync.Mutex
	sent    []mail.Message
	sendErr error
	gate    chan struct{}
}

func (m *fakeMailer) Available() bool { return true }

// Send records msg and returns the configured send error. A gated
// mailer holds the message until the gate opens.
func (m *fakeMailer) Send(_ context.Context, msg mail.Message) error {
	m.mu.Lock()
	gate := m.gate
	m.mu.Unlock()

	if gate != nil {
		select {
		case <-gate:
		case <-time.After(2 * time.Second):
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return m.sendErr
}

// Gate holds every later Send and returns the call that opens the gate.
// A held send gives up after two seconds.
func (m *fakeMailer) Gate() func() {
	m.mu.Lock()
	defer m.mu.Unlock()

	gate := make(chan struct{})
	m.gate = gate
	return sync.OnceFunc(func() { close(gate) })
}

// FailWith makes every later Send return err. A nil error restores
// plain capturing.
func (m *fakeMailer) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendErr = err
}

// Sent returns every message captured so far.
func (m *fakeMailer) Sent() []mail.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mail.Message(nil), m.sent...)
}

// WaitForSent waits for n captured messages and returns every message
// captured by then. The test fails when they do not arrive within two
// seconds.
func (m *fakeMailer) WaitForSent(t *testing.T, n int) []mail.Message {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		sent := m.Sent()
		if len(sent) >= n {
			return sent
		}
		if time.Now().After(deadline) {
			t.Fatalf("captured %d messages, want %d", len(sent), n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Reset clears the captured messages.
func (m *fakeMailer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = nil
}
