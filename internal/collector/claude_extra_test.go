package collector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseClaudeFileUnkeyedEvents(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	content := `{"type":"assistant","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":5}},"timestamp":"2026-04-12T04:48:20.228Z"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := parseClaudeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 unkeyed event, got %d", len(events))
	}
	if events[0].InputTokens != 10 || events[0].OutputTokens != 5 {
		t.Fatalf("unexpected token counts: %+v", events[0])
	}
}

func TestParseClaudeFileSkipsZeroTokens(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	content := `{"type":"assistant","message":{"model":"claude-sonnet-4-6","id":"msg_1","usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0}},"requestId":"req_1","timestamp":"2026-04-12T04:48:20.228Z"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := parseClaudeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events for zero tokens, got %d", len(events))
	}
}

func TestParseClaudeFileKeyedEventsGetContentBasedID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	line1 := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4-6",
			"id":    "msg_1",
			"usage": map[string]any{
				"input_tokens":                10,
				"cache_creation_input_tokens": 0,
				"cache_read_input_tokens":     0,
				"output_tokens":               5,
			},
		},
		"requestId": "req_1",
		"timestamp": "2026-04-12T04:48:20.228Z",
	}
	line2 := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": "claude-sonnet-4-6",
			"id":    "msg_1",
			"usage": map[string]any{
				"input_tokens":                10,
				"cache_creation_input_tokens": 0,
				"cache_read_input_tokens":     0,
				"output_tokens":               5,
			},
		},
		"requestId": "req_2",
		"timestamp": "2026-04-12T04:48:20.228Z",
	}

	data, _ := json.Marshal(line1)
	data = append(data, '\n')
	data2, _ := json.Marshal(line2)
	data = append(data, data2...)
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := parseClaudeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	lineID := hashID("claude", path, "1")
	for _, ev := range events {
		if ev.ID == lineID {
			t.Fatal("keyed event should not have line-number-based ID")
		}
	}
}

func TestParseClaudeFileNonAssistantFiltered(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	content := `{"type":"human","message":{"model":"claude-sonnet-4-6","id":"msg_1","usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":5}},"requestId":"req_1","timestamp":"2026-04-12T04:48:20.228Z"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := parseClaudeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events for non-assistant type, got %d", len(events))
	}
}
