package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var errConfigSkipped = errors.New("config skipped")

//go:embed all:embedded/defaults
var embeddedDefaults embed.FS

type InstallOptions struct {
	AssumeYes      bool
	NonInteractive bool
	BackupExisting bool
}

type Installer struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
	Options InstallOptions
}

func (i Installer) Execute(plan InstallPlan) error {
	report := RunReport{StartedAt: time.Now().UTC(), Plan: plan}
	for _, action := range plan.Actions {
		result := ActionResult{Action: action, Status: "ok"}
		if err := i.executeAction(plan, action); err != nil {
			result.Status = "error"
			result.Error = err.Error()
			report.Results = append(report.Results, result)
			report.FinishedAt = time.Now().UTC()
			_ = WriteRunReport(report)
			return err
		}
		report.Results = append(report.Results, result)
	}
	report.FinishedAt = time.Now().UTC()
	return WriteRunReport(report)
}

func (i Installer) executeAction(plan InstallPlan, action PlanAction) error {
	switch action.Kind {
	case "install_packages":
		pm, err := NewPackageManager(plan.Host.Package, plan.Host.RunningAsRoot, plan.Host.HasSudo)
		if err != nil {
			return err
		}
		missing := []string{}
		for _, pkg := range action.Packages {
			if !pm.IsInstalled(pkg) {
				missing = append(missing, pkg)
			}
		}
		if len(missing) == 0 {
			fmt.Fprintln(i.Stdout, "Packages already installed.")
			return nil
		}
		fmt.Fprintf(i.Stdout, "Installing packages: %s\n", strings.Join(missing, ", "))
		return pm.Install(missing, i.Options.AssumeYes)
	case "apply_config":
		source, err := ResolveConfigSource(plan.Options.DotfilesURL)
		if err != nil {
			return err
		}
		return ApplyConfigSource(source, plan.Profile.ConfigProfile, ConfigApplyOptions{
			Stdout:         i.Stdout,
			Stderr:         i.Stderr,
			Stdin:          i.Stdin,
			NonInteractive: i.Options.NonInteractive,
			BackupExisting: i.Options.BackupExisting,
		})
	case "source_build":
		fmt.Fprintf(i.Stdout, "Building from source: %s\n", action.Name)
		return BuildFromSource(action.Name)
	case "change_shell":
		return ChangeShellToZSH(plan.Host)
	default:
		return fmt.Errorf("unknown action kind %q", action.Kind)
	}
}

type ConfigSource struct {
	Root     fs.FS
	BasePath string
	Name     string
}

type ConfigApplyOptions struct {
	Stdout         io.Writer
	Stderr         io.Writer
	Stdin          io.Reader
	NonInteractive bool
	BackupExisting bool
	applyAll       conflictDecision
}

func ResolveConfigSource(dotfilesURL string) (ConfigSource, error) {
	if dotfilesURL == "" {
		return ConfigSource{Root: embeddedDefaults, BasePath: "embedded/defaults", Name: "embedded defaults"}, nil
	}
	target := filepath.Join(defaultHomeDir(), ".cache", "ftsh", "dotfiles")
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return ConfigSource{}, err
		}
		if err := exec.Command("git", "clone", dotfilesURL, target).Run(); err != nil {
			return ConfigSource{}, fmt.Errorf("clone dotfiles: %w", err)
		}
	} else {
		cmd := exec.Command("git", "-C", target, "pull", "--ff-only")
		if err := cmd.Run(); err != nil {
			return ConfigSource{}, fmt.Errorf("update dotfiles: %w", err)
		}
	}
	return ConfigSource{Root: os.DirFS(target), BasePath: ".", Name: dotfilesURL}, nil
}

func ApplyConfigSource(source ConfigSource, profile string, opts ConfigApplyOptions) error {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Stdin == nil {
		opts.Stdin = strings.NewReader("")
	}
	entries, err := LoadManifest(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entryApplies(entry, profile) {
			continue
		}
		if err := applyConfigEntry(source, entry, &opts); err != nil {
			if errors.Is(err, errConfigSkipped) {
				continue
			}
			return err
		}
	}
	return nil
}

func LoadManifest(source ConfigSource) ([]ConfigEntry, error) {
	candidates := []string{
		filepath.ToSlash(filepath.Join(source.BasePath, "manifest.json")),
		filepath.ToSlash(filepath.Join(source.BasePath, ".ftsh", "manifest.json")),
		filepath.ToSlash(filepath.Join(source.BasePath, "ftsh-manifest.json")),
	}
	var data []byte
	var err error
	for _, candidate := range candidates {
		data, err = fs.ReadFile(source.Root, candidate)
		if err == nil {
			var entries []ConfigEntry
			if err := json.Unmarshal(data, &entries); err != nil {
				return nil, err
			}
			return entries, nil
		}
	}
	return nil, fmt.Errorf("no manifest found in %s", source.Name)
}

func applyConfigEntry(source ConfigSource, entry ConfigEntry, opts *ConfigApplyOptions) error {
	target := expandHome(entry.Target)
	if target == "" {
		return fmt.Errorf("empty target for source %q", entry.Source)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	sourcePath := filepath.ToSlash(filepath.Join(source.BasePath, entry.Source))
	switch entry.Mode {
	case "", "copy":
		data, err := fs.ReadFile(source.Root, sourcePath)
		if err != nil {
			return err
		}
		same, err := fileContentEquals(target, data)
		if err == nil && same {
			fmt.Fprintf(opts.Stdout, "Config already current: %s\n", target)
			return nil
		}
		if err := handleConflict(target, opts); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	case "copy_dir":
		return copyConfigDir(source.Root, sourcePath, target, opts)
	case "link":
		realSource := sourcePath
		if source.BasePath == "." {
			realSource = filepath.Join(defaultHomeDir(), ".cache", "ftsh", "dotfiles", entry.Source)
		}
		if err := handleConflict(target, opts); err != nil {
			return err
		}
		return os.Symlink(realSource, target)
	case "git_clone":
		return applyGitClone(entry.Source, target, opts)
	default:
		return fmt.Errorf("unsupported config mode %q for %s", entry.Mode, entry.Target)
	}
}

func copyConfigDir(root fs.FS, sourcePath string, targetRoot string, opts *ConfigApplyOptions) error {
	return fs.WalkDir(root, sourcePath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(targetRoot, 0o755)
		}
		target := filepath.Join(targetRoot, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(root, path)
		if err != nil {
			return err
		}
		same, err := fileContentEquals(target, data)
		if err == nil && same {
			fmt.Fprintf(opts.Stdout, "Config already current: %s\n", target)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := handleConflict(target, opts); err != nil {
			if errors.Is(err, errConfigSkipped) {
				return nil
			}
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func applyGitClone(url string, target string, opts *ConfigApplyOptions) error {
	if url == "" {
		return errors.New("git_clone source URL is empty")
	}
	if _, err := os.Stat(filepath.Join(target, ".git")); err == nil {
		fmt.Fprintf(opts.Stdout, "Updating git config dependency: %s\n", target)
		return runInteractiveCommand("git", "-C", target, "pull", "--ff-only")
	}
	if _, err := os.Stat(target); err == nil {
		if err := handleConflict(target, opts); err != nil {
			if errors.Is(err, errConfigSkipped) {
				return nil
			}
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "Cloning git config dependency: %s\n", target)
	return runInteractiveCommand("git", "clone", "--depth", "1", url, target)
}

func entryApplies(entry ConfigEntry, profile string) bool {
	if len(entry.Profiles) == 0 {
		return true
	}
	for _, allowed := range entry.Profiles {
		if allowed == profile {
			return true
		}
	}
	return false
}

func fileContentEquals(path string, data []byte) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return string(existing) == string(data), nil
}

type conflictDecision int

const (
	conflictAsk conflictDecision = iota
	conflictBackupReplace
	conflictSkip
	conflictAbort
)

func handleConflict(target string, opts *ConfigApplyOptions) error {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if opts.applyAll == conflictSkip {
		return errSkipFile(target)
	}
	decision := opts.applyAll
	if decision == conflictAsk {
		decision = chooseConflict(target, opts)
	}
	switch decision {
	case conflictSkip:
		return errSkipFile(target)
	case conflictAbort:
		return fmt.Errorf("aborted while handling %s", target)
	case conflictBackupReplace:
		if opts.BackupExisting {
			backup, err := BackupPath(target)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return err
			}
			if err := os.Rename(target, backup); err != nil {
				return err
			}
			fmt.Fprintf(opts.Stdout, "Backed up %s to %s\n", target, backup)
			return nil
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory and --no-backup replacement is not supported", target)
		}
		return os.Remove(target)
	default:
		return nil
	}
}

func chooseConflict(target string, opts *ConfigApplyOptions) conflictDecision {
	if opts.NonInteractive {
		return conflictBackupReplace
	}
	fmt.Fprintf(opts.Stdout, "Config exists: %s\n", target)
	fmt.Fprint(opts.Stdout, "Choose: [b]ackup replace, backup [a]ll, [s]kip, s[k]ip all, [q]uit: ")
	reader := bufio.NewReader(opts.Stdin)
	answer, _ := reader.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "a":
		opts.applyAll = conflictBackupReplace
		return conflictBackupReplace
	case "s":
		return conflictSkip
	case "k":
		opts.applyAll = conflictSkip
		return conflictSkip
	case "q":
		return conflictAbort
	default:
		return conflictBackupReplace
	}
}

func errSkipFile(path string) error {
	return fmt.Errorf("%w: %s", errConfigSkipped, path)
}

func BackupPath(target string) (string, error) {
	rel, err := filepath.Rel(defaultHomeDir(), target)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = strings.TrimPrefix(filepath.Clean(target), string(filepath.Separator))
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	return filepath.Join(defaultHomeDir(), ".ftsh", "backups", stamp, rel), nil
}

func expandHome(path string) string {
	if path == "~" {
		return defaultHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(defaultHomeDir(), strings.TrimPrefix(path, "~/"))
	}
	return path
}

func WriteDefaultConfig(path string) error {
	data := []byte(`{
  "profile": "server",
  "dotfiles": "",
  "allow_source_builds": false
}
`)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	return os.WriteFile(path, data, 0o644)
}

func WriteRunReport(report RunReport) error {
	dir := filepath.Join(defaultHomeDir(), ".local", "state", "ftsh", "runs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, report.StartedAt.Format("20060102T150405Z")+".json")
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func ChangeShellToZSH(host HostInfo) error {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		return nil
	}
	if host.Shell == zsh || strings.HasSuffix(host.Shell, "/zsh") {
		return nil
	}
	return exec.Command("chsh", "-s", zsh).Run()
}
