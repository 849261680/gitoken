package collector

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestParseOpenCodeDBSkipsNonAssistantRole(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, stmt := range []string{
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	userMsg := `{"role":"user","modelID":"test","providerID":"test"}`
	assistantMsg := `{"role":"assistant","modelID":"deepseek/v3","providerID":"deepseek"}`
	stepFinish := `{"type":"step-finish","tokens":{"total":100,"input":80,"output":20,"reasoning":0,"cache":{"write":0,"read":0}}}`

	db.Exec(`INSERT INTO message(id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?)`,
		"msg_user", "ses_1", int64(1000), int64(1000), userMsg)
	db.Exec(`INSERT INTO part(id, message_id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?, ?)`,
		"prt_user", "msg_user", "ses_1", int64(1000), int64(1000), stepFinish)

	db.Exec(`INSERT INTO message(id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?)`,
		"msg_asst", "ses_1", int64(2000), int64(2000), assistantMsg)
	db.Exec(`INSERT INTO part(id, message_id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?, ?)`,
		"prt_asst", "msg_asst", "ses_1", int64(2000), int64(2000), stepFinish)

	events, err := parseOpenCodeDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (user role filtered), got %d", len(events))
	}
	if events[0].Model != "deepseek/v3" {
		t.Fatalf("expected model deepseek/v3, got %s", events[0].Model)
	}
}

func TestParseOpenCodeDBTotalZeroFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, stmt := range []string{
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	messageJSON := `{"role":"assistant","modelID":"test-model","providerID":"test"}`
	partJSON := `{"type":"step-finish","tokens":{"total":0,"input":50,"output":30,"reasoning":0,"cache":{"write":10,"read":20}}}`

	db.Exec(`INSERT INTO message(id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?)`,
		"msg_1", "ses_1", int64(1776069954512), int64(1776069975877), messageJSON)
	db.Exec(`INSERT INTO part(id, message_id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?, ?)`,
		"prt_1", "msg_1", "ses_1", int64(1776069975877), int64(1776069975877), partJSON)

	events, err := parseOpenCodeDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	expectedTotal := 50 + 20 + 10 + 30
	if events[0].TotalTokens != expectedTotal {
		t.Fatalf("expected total %d (sum of components), got %d", expectedTotal, events[0].TotalTokens)
	}
}

func TestParseOpenCodeDBSkipsNonStepFinish(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, stmt := range []string{
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	messageJSON := `{"role":"assistant","modelID":"test","providerID":"test"}`
	textPart := `{"type":"text","text":"hello","tokens":{"total":0,"input":0,"output":0,"reasoning":0,"cache":{"write":0,"read":0}}}`

	db.Exec(`INSERT INTO message(id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?)`,
		"msg_1", "ses_1", int64(1000), int64(1000), messageJSON)
	db.Exec(`INSERT INTO part(id, message_id, session_id, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?, ?)`,
		"prt_1", "msg_1", "ses_1", int64(1000), int64(1000), textPart)

	events, err := parseOpenCodeDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events (text part filtered), got %d", len(events))
	}
}

func TestOpenCodeDBPathUsesOpenCodeDataDir(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv("OPENCODE_DATA_DIR", customDir)
	t.Setenv("XDG_DATA_HOME", "")

	path, err := opencodeDBPath()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(customDir, "opencode.db")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}
