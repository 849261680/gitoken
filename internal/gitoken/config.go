package gitoken

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/849261680/token-heatmap/internal/store"
)

// Config holds persisted setup state so subsequent commands can resolve the
// profile repo without needing --profile-repo-dir on every invocation.
type Config struct {
	GitHubUsername string `json:"github_username"`
	ProfileRepoDir string `json:"profile_repo_dir"`
	CreatedAt      string `json:"created_at"`
}

func configDir() (string, error) {
	dbPath, err := store.DefaultDBPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(dbPath), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig reads ~/.tokenheat/config.json. Returns an empty Config if the
// file does not exist.
func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// SaveConfig writes cfg to ~/.tokenheat/config.json, creating the directory
// if needed.
func SaveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func runConfig(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing config subcommand: use 'show' or 'set'")
	}
	switch args[0] {
	case "show":
		return runConfigShow(args[1:])
	case "set":
		return runConfigSet(args[1:])
	default:
		return fmt.Errorf("unknown config subcommand %q", args[0])
	}
}

func runConfigShow(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("config show does not accept arguments")
	}
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.GitHubUsername == "" && cfg.ProfileRepoDir == "" {
		fmt.Println("no config found (run 'tokenheat init' first)")
		return nil
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runConfigSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: tokenheat config set <key> <value>\nkeys: github_username, profile_repo_dir")
	}
	key := args[0]
	value := args[1]

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	switch key {
	case "github_username":
		cfg.GitHubUsername = value
	case "profile_repo_dir":
		cfg.ProfileRepoDir = value
	default:
		return fmt.Errorf("unknown key %q (valid: github_username, profile_repo_dir)", key)
	}

	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("config updated: %s = %s\n", key, value)
	return nil
}
