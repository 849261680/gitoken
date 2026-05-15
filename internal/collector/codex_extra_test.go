package collector

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCodexFileLastTokenUsage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	content := `{"timestamp":"2025-09-16T14:47:59.240Z","type":"turn_context","payload":{"model":"gpt-5-codex"}}
{"timestamp":"2025-09-16T14:48:02.551Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":80,"output_tokens":10}}}}
{"timestamp":"2025-09-16T14:49:02.551Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":40,"cached_input_tokens":20,"output_tokens":5}}}}
` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexFile(path, codexMetadata{}, func(string, time.Time) codexTotals { return codexTotals{} })
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].InputTokens != 100 || events[0].CacheReadTokens != 80 || events[0].OutputTokens != 10 {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if events[1].InputTokens != 40 || events[1].CacheReadTokens != 20 || events[1].OutputTokens != 5 {
		t.Fatalf("unexpected second event: %+v", events[1])
	}
}

func TestParseCodexFileSkipsZeroDeltas(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	content := `{"timestamp":"2025-09-16T14:47:59.240Z","type":"turn_context","payload":{"model":"gpt-5-codex"}}
{"timestamp":"2025-09-16T14:48:02.551Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":80,"output_tokens":10}}}}
{"timestamp":"2025-09-16T14:49:02.551Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":80,"output_tokens":10}}}}
` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexFile(path, codexMetadata{}, func(string, time.Time) codexTotals { return codexTotals{} })
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (zero delta skipped), got %d", len(events))
	}
}

func TestParseCodexFileForkInheritance(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	parentPath := filepath.Join(dir, "parent-session-abc-1234-5678-9abc-def0.jsonl")
	childPath := filepath.Join(dir, "child-session-abc-1234-5678-9abc-def1.jsonl")

	parentContent := `{"timestamp":"2025-09-16T14:47:59.240Z","type":"session_meta","payload":{"session_id":"ses-parent"}}
{"timestamp":"2025-09-16T14:48:02.551Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":50,"output_tokens":20}}}}
` + "\n"
	if err := os.WriteFile(parentPath, []byte(parentContent), 0o644); err != nil {
		t.Fatal(err)
	}

	childContent := `{"timestamp":"2025-09-16T14:48:00.000Z","type":"session_meta","payload":{"session_id":"ses-child","forked_from_id":"ses-parent","timestamp":"2025-09-16T14:48:02.551Z"}}
{"timestamp":"2025-09-16T14:49:02.551Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":150,"cached_input_tokens":70,"output_tokens":30}}}}
` + "\n"
	if err := os.WriteFile(childPath, []byte(childContent), 0o644); err != nil {
		t.Fatal(err)
	}

	resolveInherited := func(sessionID string, cutoff time.Time) codexTotals {
		return codexTotals{Input: 100, Cached: 50, Output: 20}
	}

	meta := codexMetadata{
		SessionID:     "ses-child",
		ForkedFromID:  "ses-parent",
		ForkTimestamp: "2025-09-16T14:48:02.551Z",
	}

	events, err := parseCodexFile(childPath, meta, resolveInherited)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].InputTokens != 50 {
		t.Fatalf("expected input delta 50 (150-100), got %d", events[0].InputTokens)
	}
	if events[0].CacheReadTokens != 20 {
		t.Fatalf("expected cached delta 20 (70-50), got %d", events[0].CacheReadTokens)
	}
	if events[0].OutputTokens != 10 {
		t.Fatalf("expected output delta 10 (30-20), got %d", events[0].OutputTokens)
	}
}

func TestReadCodexMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample-session-abc-1234-5678-9abc-def0.jsonl")
	content := `{"timestamp":"2025-09-16T14:47:59.240Z","type":"session_meta","payload":{"session_id":"ses-123","forked_from_id":"ses-parent","timestamp":"2025-09-16T14:47:00.000Z"}}
` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	meta, err := readCodexMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.SessionID != "ses-123" {
		t.Fatalf("expected session_id ses-123, got %q", meta.SessionID)
	}
	if meta.ForkedFromID != "ses-parent" {
		t.Fatalf("expected forked_from_id ses-parent, got %q", meta.ForkedFromID)
	}
}

func TestReadCodexMetadataFallsBackToPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "some-prefix-abc-1234-5678-9abc-def0.jsonl")
	content := `{"timestamp":"2025-09-16T14:47:59.240Z","type":"event_msg","payload":{}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	meta, err := readCodexMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if meta.SessionID != "abc-1234-5678-9abc-def0" {
		t.Fatalf("expected path-based session ID, got %q", meta.SessionID)
	}
}

func TestReadCodexSnapshots(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	content := `{"timestamp":"2025-09-16T14:48:02.551Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":80,"output_tokens":10}}}}
{"timestamp":"2025-09-16T14:49:02.551Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":200,"cached_input_tokens":100,"output_tokens":20}}}}
` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshots, err := readCodexSnapshots(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots[0].Totals.Input != 100 {
		t.Fatalf("expected first snapshot input 100, got %d", snapshots[0].Totals.Input)
	}
	if snapshots[1].Totals.Input != 200 {
		t.Fatalf("expected second snapshot input 200, got %d", snapshots[1].Totals.Input)
	}
}

func TestParseCodexSessionIDFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/tmp/codex/sessions/session-abc-1234-5678-9abc-def0.jsonl", "abc-1234-5678-9abc-def0"},
		{"/tmp/short.jsonl", ""},
	}
	for _, tt := range tests {
		got := parseCodexSessionIDFromPath(tt.path)
		if got != tt.expected {
			t.Errorf("parseCodexSessionIDFromPath(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestHashIDDeterministic(t *testing.T) {
	a := hashID("a", "b", "c")
	b := hashID("a", "b", "c")
	if a != b {
		t.Fatal("hashID should be deterministic")
	}
	if len(a) != 40 {
		t.Fatalf("expected 40-char hex SHA1, got %d chars", len(a))
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "hello"); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
	if got := firstNonEmpty("", "  "); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestAsInt(t *testing.T) {
	if got := asInt(float64(42)); got != 42 {
		t.Fatalf("expected 42 from float64, got %d", got)
	}
	if got := asInt(42); got != 42 {
		t.Fatalf("expected 42 from int, got %d", got)
	}
	if got := asInt(nil); got != 0 {
		t.Fatalf("expected 0 from nil, got %d", got)
	}
}

func TestAsString(t *testing.T) {
	if got := asString("hello"); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
	if got := asString(42); got != "" {
		t.Fatalf("expected empty for non-string, got %q", got)
	}
}

func TestParseTimestamp(t *testing.T) {
	ts, ok := parseTimestamp("2025-09-16T14:48:02.551Z")
	if !ok {
		t.Fatal("expected valid timestamp")
	}
	if ts.Year() != 2025 {
		t.Fatalf("expected 2025, got %d", ts.Year())
	}

	_, ok = parseTimestamp("")
	if ok {
		t.Fatal("expected empty string to fail")
	}

	_, ok = parseTimestamp("not-a-timestamp")
	if ok {
		t.Fatal("expected invalid timestamp to fail")
	}
}

func TestMinIntMaxInt(t *testing.T) {
	if minInt(3, 5) != 3 {
		t.Fatal("minInt(3,5) should be 3")
	}
	if maxInt(3, 5) != 5 {
		t.Fatal("maxInt(3,5) should be 5")
	}
}
