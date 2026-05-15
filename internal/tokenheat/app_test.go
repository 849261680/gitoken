package tokenheat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/849261680/token-heatmap/internal/model"
)

func TestParseProviders(t *testing.T) {
	tests := []struct {
		input    string
		expected []model.Provider
		wantErr  bool
	}{
		{"all", []model.Provider{model.ProviderCodex, model.ProviderClaude, model.ProviderOpenCode}, false},
		{"codex", []model.Provider{model.ProviderCodex}, false},
		{"claude", []model.Provider{model.ProviderClaude}, false},
		{"opencode", []model.Provider{model.ProviderOpenCode}, false},
		{"bad", nil, true},
	}
	for _, tt := range tests {
		got, err := parseProviders(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseProviders(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if len(got) != len(tt.expected) {
			t.Errorf("parseProviders(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := Config{
		GitHubUsername: "testuser",
		ProfileRepoDir: "/tmp/repo",
		CreatedAt:      "2026-01-01",
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(dir, ".tokenheat", "config.json")
	if _, err := os.Stat(configFile); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GitHubUsername != cfg.GitHubUsername {
		t.Fatalf("expected username %q, got %q", cfg.GitHubUsername, loaded.GitHubUsername)
	}
	if loaded.ProfileRepoDir != cfg.ProfileRepoDir {
		t.Fatalf("expected profile dir %q, got %q", cfg.ProfileRepoDir, loaded.ProfileRepoDir)
	}
	if loaded.CreatedAt != cfg.CreatedAt {
		t.Fatalf("expected created_at %q, got %q", cfg.CreatedAt, loaded.CreatedAt)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GitHubUsername != "" || cfg.ProfileRepoDir != "" {
		t.Fatal("expected empty config for missing file")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := Run([]string{"unknown-cmd"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}
