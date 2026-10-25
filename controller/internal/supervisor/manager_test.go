package supervisor

import (
	"strings"
	"testing"
)

func TestStreamLogsPreservesFatalErrorForStatus(t *testing.T) {
	m := &Manager{
		logs:        NewRingBuffer(10),
		subscribers: make(map[chan LogLine]struct{}),
	}

	m.streamLogs(strings.NewReader(`FATAL[0015] start service: initialize rule-set[tiktok]: timeout`+"\n"), "stderr")

	status := m.Status()
	if status.FatalError == "" {
		t.Fatal("expected fatal error to be exposed in supervisor status")
	}
	if status.LastError != status.FatalError {
		t.Fatalf("expected last_error to preserve fatal error, got %q", status.LastError)
	}
}
