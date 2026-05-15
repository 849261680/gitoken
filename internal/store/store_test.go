package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/849261680/token-heatmap/internal/model"

	_ "modernc.org/sqlite"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	t.Setenv("HOME", dir)
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestOpenCreatesTables(t *testing.T) {
	st := openTestStore(t)

	for _, table := range []string{"usage_events", "file_states"} {
		var name string
		err := st.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s not found: %v", table, err)
		}
	}
}

func TestOpenCreatesIndexes(t *testing.T) {
	st := openTestStore(t)

	for _, idx := range []string{"idx_usage_events_day_provider", "idx_usage_events_source_file"} {
		var name string
		err := st.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&name)
		if err != nil {
			t.Fatalf("index %s not found: %v", idx, err)
		}
	}
}

func TestOpenIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := filepath.Join(dir, "test.db")

	st1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st1.Close()

	st2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st2.Close()
}

func TestReplaceFileEventsInsert(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	events := []model.UsageEvent{
		{
			ID: "ev1", Provider: model.ProviderCodex, SourceFile: "/tmp/a.jsonl",
			EventTime: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
			Day: "2026-05-15", Model: "gpt-5",
			InputTokens: 100, CacheReadTokens: 50, CacheWriteTokens: 10, OutputTokens: 30, TotalTokens: 190,
		},
	}

	err := st.ReplaceFileEvents(ctx, model.ProviderCodex, "/tmp/a.jsonl", 1024, time.Now(), events)
	if err != nil {
		t.Fatal(err)
	}

	count, err := st.tableCount(ctx, "usage_events")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
}

func TestReplaceFileEventsReplaces(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	path := "/tmp/a.jsonl"

	events1 := []model.UsageEvent{
		{ID: "ev1", Provider: model.ProviderCodex, SourceFile: path, EventTime: time.Now(), Day: "2026-05-15", Model: "gpt-5", InputTokens: 100, TotalTokens: 100},
		{ID: "ev2", Provider: model.ProviderCodex, SourceFile: path, EventTime: time.Now(), Day: "2026-05-15", Model: "gpt-5", InputTokens: 200, TotalTokens: 200},
	}
	err := st.ReplaceFileEvents(ctx, model.ProviderCodex, path, 1024, time.Now(), events1)
	if err != nil {
		t.Fatal(err)
	}

	events2 := []model.UsageEvent{
		{ID: "ev3", Provider: model.ProviderCodex, SourceFile: path, EventTime: time.Now(), Day: "2026-05-15", Model: "gpt-5", InputTokens: 300, TotalTokens: 300},
	}
	err = st.ReplaceFileEvents(ctx, model.ProviderCodex, path, 2048, time.Now(), events2)
	if err != nil {
		t.Fatal(err)
	}

	count, err := st.tableCount(ctx, "usage_events")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event after replace, got %d", count)
	}
}

func TestReplaceFileEventsUpsertsFileState(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	modTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	err := st.ReplaceFileEvents(ctx, model.ProviderClaude, "/tmp/a.jsonl", 512, modTime, []model.UsageEvent{
		{ID: "ev1", Provider: model.ProviderClaude, SourceFile: "/tmp/a.jsonl", EventTime: time.Now(), Day: "2026-01-01", Model: "claude", InputTokens: 10, TotalTokens: 10},
	})
	if err != nil {
		t.Fatal(err)
	}

	state, ok, err := st.FileState(ctx, model.ProviderClaude, "/tmp/a.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected file state to be found")
	}
	if state.SizeBytes != 512 {
		t.Fatalf("expected size 512, got %d", state.SizeBytes)
	}

	modTime2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	err = st.ReplaceFileEvents(ctx, model.ProviderClaude, "/tmp/a.jsonl", 1024, modTime2, []model.UsageEvent{
		{ID: "ev2", Provider: model.ProviderClaude, SourceFile: "/tmp/a.jsonl", EventTime: time.Now(), Day: "2026-02-01", Model: "claude", InputTokens: 20, TotalTokens: 20},
	})
	if err != nil {
		t.Fatal(err)
	}

	state, ok, err = st.FileState(ctx, model.ProviderClaude, "/tmp/a.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if state.SizeBytes != 1024 {
		t.Fatalf("expected updated size 1024, got %d", state.SizeBytes)
	}
}

func TestFileStateNotFound(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	_, ok, err := st.FileState(ctx, model.ProviderCodex, "/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected not found")
	}
}

func TestDeleteMissingFiles(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	for _, p := range []string{"/tmp/a.jsonl", "/tmp/b.jsonl", "/tmp/c.jsonl"} {
		err := st.ReplaceFileEvents(ctx, model.ProviderCodex, p, 100, time.Now(), []model.UsageEvent{
			{ID: "ev_" + filepath.Base(p), Provider: model.ProviderCodex, SourceFile: p, EventTime: time.Now(), Day: "2026-05-15", Model: "gpt", InputTokens: 10, TotalTokens: 10},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	currentPaths := map[string]struct{}{
		"/tmp/a.jsonl": {},
		"/tmp/c.jsonl": {},
	}
	err := st.DeleteMissingFiles(ctx, model.ProviderCodex, currentPaths)
	if err != nil {
		t.Fatal(err)
	}

	count, err := st.tableCount(ctx, "usage_events")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 events after delete, got %d", count)
	}

	_, ok, err := st.FileState(ctx, model.ProviderCodex, "/tmp/b.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected /tmp/b.jsonl file state to be deleted")
	}
}

func TestDailyUsageSince(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	insertEvents := []struct {
		provider   model.Provider
		sourceFile string
		event      model.UsageEvent
	}{
		{model.ProviderCodex, "/tmp/a.jsonl", model.UsageEvent{ID: "ev1", Provider: model.ProviderCodex, SourceFile: "/tmp/a.jsonl", EventTime: time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC), Day: "2026-05-14", Model: "gpt", InputTokens: 100, OutputTokens: 50, TotalTokens: 150}},
		{model.ProviderCodex, "/tmp/b.jsonl", model.UsageEvent{ID: "ev2", Provider: model.ProviderCodex, SourceFile: "/tmp/b.jsonl", EventTime: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC), Day: "2026-05-15", Model: "gpt", InputTokens: 200, OutputTokens: 30, TotalTokens: 230}},
		{model.ProviderClaude, "/tmp/c.jsonl", model.UsageEvent{ID: "ev3", Provider: model.ProviderClaude, SourceFile: "/tmp/c.jsonl", EventTime: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC), Day: "2026-05-15", Model: "claude", InputTokens: 50, OutputTokens: 10, TotalTokens: 60}},
	}
	for _, ie := range insertEvents {
		err := st.ReplaceFileEvents(ctx, ie.provider, ie.sourceFile, 100, time.Now(), []model.UsageEvent{ie.event})
		if err != nil {
			t.Fatal(err)
		}
	}

	rows, err := st.DailyUsageSince(ctx, "2026-05-14")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].Day != "2026-05-14" || rows[0].Provider != model.ProviderCodex || rows[0].InputTokens != 100 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
}

func TestDailyUsageForDay(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	err := st.ReplaceFileEvents(ctx, model.ProviderClaude, "/tmp/a.jsonl", 100, time.Now(), []model.UsageEvent{
		{ID: "ev1", Provider: model.ProviderClaude, SourceFile: "/tmp/a.jsonl", EventTime: time.Now(), Day: "2026-05-15", Model: "claude", InputTokens: 100, CacheReadTokens: 20, OutputTokens: 30, TotalTokens: 150},
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := st.DailyUsageForDay(ctx, "2026-05-15")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].InputTokens != 100 || rows[0].CacheReadTokens != 20 || rows[0].OutputTokens != 30 {
		t.Fatalf("unexpected token counts: %+v", rows[0])
	}

	rows, err = st.DailyUsageForDay(ctx, "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for absent day, got %d", len(rows))
	}
}

func TestMigrateLegacyData(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	legacyDir := filepath.Join(dir, ".gitoken")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(legacyDir, "gitoken.db")

	legacyDB, err := sql.Open("sqlite", legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS usage_events (id TEXT PRIMARY KEY, provider TEXT NOT NULL, source_file TEXT NOT NULL, event_time TEXT NOT NULL, day TEXT NOT NULL, model TEXT NOT NULL, input_tokens INTEGER NOT NULL, cache_read_tokens INTEGER NOT NULL, cache_write_tokens INTEGER NOT NULL, output_tokens INTEGER NOT NULL, total_tokens INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS file_states (provider TEXT NOT NULL, path TEXT NOT NULL, size_bytes INTEGER NOT NULL, mod_unix_ms INTEGER NOT NULL, PRIMARY KEY (provider, path))`,
		`INSERT INTO usage_events VALUES ('ev1','codex','/tmp/a.jsonl','2026-01-01T00:00:00Z','2026-01-01','gpt',100,0,0,50,150)`,
		`INSERT INTO file_states VALUES ('codex','/tmp/a.jsonl',1024,1704067200000)`,
	} {
		if _, err := legacyDB.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	legacyDB.Close()

	currentPath := filepath.Join(dir, "tokenheat", "tokenheat.db")
	st, err := Open(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	count, err := st.tableCount(ctx, "usage_events")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 migrated event, got %d", count)
	}

	state, ok, err := st.FileState(ctx, model.ProviderCodex, "/tmp/a.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected migrated file state")
	}
	if state.SizeBytes != 1024 {
		t.Fatalf("expected size 1024, got %d", state.SizeBytes)
	}
}

func TestDefaultDBPath(t *testing.T) {
	path, err := DefaultDBPath()
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if filepath.Base(path) != "tokenheat.db" {
		t.Fatalf("expected base name tokenheat.db, got %s", filepath.Base(path))
	}
}
