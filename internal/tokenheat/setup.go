package tokenheat

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/849261680/token-heatmap/internal/model"
)

type initOptions struct {
	Username      string
	NoProfile     bool
	InstallSched  bool
	ProfileRepoDir string
	Sync          syncOptions
}

func runInit(args []string) error {
	opts, providers, err := parseInitOptions(args)
	if err != nil {
		return err
	}

	if opts.Username == "" && !opts.NoProfile {
		fmt.Print("detecting GitHub username... ")
		opts.Username = DetectGitHubUsername()
		if opts.Username != "" {
			fmt.Println(opts.Username)
		} else {
			fmt.Println("not found")
		}
	}

	if opts.Username != "" && !opts.NoProfile {
		profileDir, err := setupProfileRepo(opts.Username, opts.ProfileRepoDir)
		if err != nil {
			return fmt.Errorf("setup profile repo: %w", err)
		}
		opts.Sync.ProfileRepoDir = profileDir
	} else if opts.NoProfile {
		fmt.Println("skipping GitHub profile setup (--no-profile)")
	}

	fmt.Println("collecting token usage...")
	results, err := executeCollect(collectOptions{
		DBPath:    opts.Sync.Generate.DBPath,
		Providers: providers,
	})
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}
	var views []collectResultView
	for _, r := range results {
		views = append(views, collectResultView{
			Provider:      string(r.Provider),
			FilesScanned:  r.FilesScanned,
			FilesSkipped:  r.FilesSkipped,
			EventsWritten: r.EventsWritten,
		})
	}
	printCollectResults(views)

	fmt.Println("generating and syncing...")
	if err := syncGitHub(opts.Sync); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	cfg := Config{
		GitHubUsername: opts.Username,
		ProfileRepoDir: opts.Sync.ProfileRepoDir,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("saved config to ~/.tokenheat/config.json\n")

	if opts.InstallSched {
		fmt.Println("installing daily schedule...")
		schedOpts := scheduleInstallOptions{
			Time:           "00:05",
			BinaryPath:     os.Args[0],
			RepoDir:        opts.Sync.RepoDir,
			ProfileRepoDir: opts.Sync.ProfileRepoDir,
			DBPath:         opts.Sync.Generate.DBPath,
			Days:           opts.Sync.Generate.Days,
			OutputDir:      "docs",
			Remote:         opts.Sync.Remote,
			ProfileRemote:  opts.Sync.ProfileRemote,
			ProfileAsset:   opts.Sync.ProfileAsset,
			Provider:       "all",
		}
		if err := writeSchedule(schedOpts); err != nil {
			return fmt.Errorf("install schedule: %w", err)
		}
		fmt.Println("schedule installed")
	}

	fmt.Println("\nSetup complete!")
	if opts.Sync.ProfileRepoDir != "" {
		fmt.Printf("Profile heatmap: https://github.com/%s/%s\n", opts.Username, opts.Username)
	}
	return nil
}

func setupProfileRepo(username, explicitDir string) (string, error) {
	dir := explicitDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".tokenheat", "profile-repo")
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		url := fmt.Sprintf("https://github.com/%s/%s.git", username, username)
		fmt.Printf("cloning %s...\n", url)
		if err := os.RemoveAll(dir); err != nil {
			return "", fmt.Errorf("remove stale profile dir: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
			return "", err
		}
		cmd := exec.Command("git", "clone", url, dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("clone profile repo %s: %w\n%s", url, err, string(out))
		}
		fmt.Printf("cloned to %s\n", dir)
	}
	return dir, nil
}

func writeSchedule(opts scheduleInstallOptions) error {
	if err := os.MkdirAll(filepath.Dir(launchAgentPath()), 0o755); err != nil {
		return fmt.Errorf("create launch agents directory: %w", err)
	}
	if err := os.MkdirAll(scheduleLogDir(), 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	args := launchAgentProgramArgs(opts)
	plist := buildLaunchAgentPlist(scheduleTrigger{Hour: 0, Minute: 5}, args, opts.RepoDir)
	if err := os.WriteFile(launchAgentPath(), []byte(plist), 0o644); err != nil {
		return fmt.Errorf("write launch agent plist: %w", err)
	}

	_ = exec.Command("launchctl", "unload", launchAgentPath()).Run()
	return exec.Command("launchctl", "load", launchAgentPath()).Run()
}

func parseInitOptions(args []string) (initOptions, []model.Provider, error) {
	syncOpts, err := parseSyncGitHubOptions(nil)
	if err != nil {
		return initOptions{}, nil, err
	}

	opts := initOptions{
		Sync: syncOpts,
	}

	fs := newFlagSet("init")
	username := fs.String("username", "", "GitHub username (auto-detected if not set)")
	noProfile := fs.Bool("no-profile", false, "skip GitHub profile repo setup")
	schedule := fs.Bool("schedule", false, "also install macOS daily schedule")
	repoDir := fs.String("repo-dir", opts.Sync.RepoDir, "git repository directory")
	dbPath := fs.String("db", opts.Sync.Generate.DBPath, "sqlite database path")
	days := fs.Int("days", opts.Sync.Generate.Days, "number of local days to export")
	outputDir := fs.String("output-dir", "docs", "output directory relative to repo-dir")
	remote := fs.String("remote", opts.Sync.Remote, "git remote name")
	branch := fs.String("branch", "", "branch to push")
	profileRepoDir := fs.String("profile-repo-dir", "", "GitHub profile repository directory (overrides auto)")
	profileRemote := fs.String("profile-remote", opts.Sync.ProfileRemote, "git remote for profile repo")
	profileBranch := fs.String("profile-branch", "", "branch for profile repo")
	profileAsset := fs.String("profile-asset", opts.Sync.ProfileAsset, "heatmap asset path relative to profile repo")
	providerArg := fs.String("provider", "all", "all|codex|claude|opencode")
	if err := fs.Parse(args); err != nil {
		return initOptions{}, nil, err
	}

	providers, err := parseProviders(*providerArg)
	if err != nil {
		return initOptions{}, nil, err
	}

	opts.Username = *username
	opts.NoProfile = *noProfile
	opts.InstallSched = *schedule
	opts.ProfileRepoDir = *profileRepoDir
	opts.Sync.RepoDir = *repoDir
	opts.Sync.Remote = *remote
	opts.Sync.Branch = *branch
	opts.Sync.ProfileRepoDir = *profileRepoDir
	opts.Sync.ProfileRemote = *profileRemote
	opts.Sync.ProfileBranch = *profileBranch
	opts.Sync.ProfileAsset = *profileAsset
	opts.Sync.Generate.DBPath = *dbPath
	opts.Sync.Generate.Days = *days
	opts.Sync.Generate.OutputDir = resolveOutputDir(*repoDir, *outputDir)
	opts.Sync.Generate.Now = time.Now()
	return opts, providers, nil
}

// confirm prompts the user for a yes/no answer on stdin. Returns true for "y"/"yes".
func confirm(prompt string) bool {
	fmt.Printf("%s [Y/n] ", prompt)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "" || line == "y" || line == "yes"
}
