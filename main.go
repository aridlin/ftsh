package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "ftsh:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "doctor":
		return runDoctor(args[1:], stdout)
	case "profiles":
		return runProfiles(args[1:], stdout)
	case "plan":
		return runPlan(args[1:], stdout)
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout)
	case "tui":
		return runTUI(args[1:], stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, version)
		return nil
	case "help", "--help", "-h":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "FTSH - portable workstation bootstrapper")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ftsh doctor")
	fmt.Fprintln(w, "  ftsh profiles")
	fmt.Fprintln(w, "  ftsh plan --profile dev [--dotfiles URL] [--allow-source-builds]")
	fmt.Fprintln(w, "  ftsh install --profile dev [--dotfiles URL] [--yes] [--non-interactive]")
	fmt.Fprintln(w, "  ftsh config init [--path PATH]")
	fmt.Fprintln(w, "  ftsh tui")
}

func runDoctor(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("doctor does not accept positional arguments")
	}

	host := DetectHost()
	WriteDoctor(stdout, host)
	return nil
}

func runProfiles(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("profiles", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("profiles does not accept positional arguments")
	}
	for _, profile := range BuiltinProfiles() {
		fmt.Fprintf(stdout, "%-10s %s\n", profile.Name, profile.Description)
	}
	return nil
}

func runPlan(args []string, stdout io.Writer) error {
	opts, err := parsePlanFlags("plan", args)
	if err != nil {
		return err
	}

	plan, err := BuildPlan(DetectHost(), opts)
	if err != nil {
		return err
	}
	WritePlan(stdout, plan)
	return nil
}

func runInstall(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileName := fs.String("profile", "server", "profile to install")
	dotfiles := fs.String("dotfiles", "", "git URL for dotfiles")
	config := fs.String("config", "", "reserved config file path")
	allowSource := fs.Bool("allow-source-builds", false, "allow source-build actions")
	yes := fs.Bool("yes", false, "accept package-manager confirmations")
	nonInteractive := fs.Bool("non-interactive", false, "do not prompt")
	backupExisting := fs.Bool("backup-existing", true, "backup conflicting files before replacing")
	noBackup := fs.Bool("no-backup", false, "replace conflicting files without backups")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("install does not accept positional arguments")
	}
	if *config != "" {
		return errors.New("--config is reserved for a future declarative config file")
	}

	opts := PlanOptions{
		ProfileName:       *profileName,
		DotfilesURL:       *dotfiles,
		AllowSourceBuilds: *allowSource,
	}
	plan, err := BuildPlan(DetectHost(), opts)
	if err != nil {
		return err
	}
	WritePlan(stdout, plan)
	if len(plan.Warnings) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Warnings must be resolved or accepted before install continues.")
	}

	installer := Installer{
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  os.Stdin,
		Options: InstallOptions{
			AssumeYes:      *yes,
			NonInteractive: *nonInteractive,
			BackupExisting: *backupExisting && !*noBackup,
		},
	}
	return installer.Execute(plan)
}

func runConfig(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("config requires a subcommand")
	}
	switch args[0] {
	case "init":
		fs := flag.NewFlagSet("config init", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		defaultPath := filepath.Join(defaultHomeDir(), ".config", "ftsh", "config.json")
		path := fs.String("path", defaultPath, "config path")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return errors.New("config init does not accept positional arguments")
		}
		if err := WriteDefaultConfig(*path); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Wrote %s\n", *path)
		return nil
	default:
		return fmt.Errorf("unknown config subcommand %q", args[0])
	}
}

func parsePlanFlags(name string, args []string) (PlanOptions, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileName := fs.String("profile", "server", "profile to install")
	dotfiles := fs.String("dotfiles", "", "git URL for dotfiles")
	config := fs.String("config", "", "reserved config file path")
	allowSource := fs.Bool("allow-source-builds", false, "allow source-build actions")
	if err := fs.Parse(args); err != nil {
		return PlanOptions{}, err
	}
	if fs.NArg() != 0 {
		return PlanOptions{}, fmt.Errorf("%s does not accept positional arguments", name)
	}
	if *config != "" {
		return PlanOptions{}, errors.New("--config is reserved for a future declarative config file")
	}
	return PlanOptions{
		ProfileName:       *profileName,
		DotfilesURL:       *dotfiles,
		AllowSourceBuilds: *allowSource,
	}, nil
}

func runTUI(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileName := fs.String("profile", "server", "initial profile")
	dotfiles := fs.String("dotfiles", "", "git URL for dotfiles")
	allowSource := fs.Bool("allow-source-builds", false, "allow source-build actions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("tui does not accept positional arguments")
	}

	state := tuiState{
		profileName:       *profileName,
		dotfilesURL:       *dotfiles,
		allowSourceBuilds: *allowSource,
		reader:            bufio.NewReader(os.Stdin),
		stdout:            stdout,
		stderr:            stderr,
	}
	return state.run()
}

type tuiState struct {
	profileName       string
	dotfilesURL       string
	allowSourceBuilds bool
	reader            *bufio.Reader
	stdout            io.Writer
	stderr            io.Writer
}

func (s *tuiState) run() error {
	for {
		s.clear()
		fmt.Fprintln(s.stdout, "FTSH")
		fmt.Fprintln(s.stdout, "----")
		fmt.Fprintf(s.stdout, "Profile: %s\n", s.profileName)
		fmt.Fprintf(s.stdout, "Dotfiles: %s\n", emptyAsNone(s.dotfilesURL))
		fmt.Fprintf(s.stdout, "Source builds: %t\n", s.allowSourceBuilds)
		fmt.Fprintln(s.stdout)
		fmt.Fprintln(s.stdout, "1) Doctor")
		fmt.Fprintln(s.stdout, "2) Preview plan")
		fmt.Fprintln(s.stdout, "3) Change profile")
		fmt.Fprintln(s.stdout, "4) Set dotfiles URL")
		fmt.Fprintln(s.stdout, "5) Toggle source builds")
		fmt.Fprintln(s.stdout, "6) Install")
		fmt.Fprintln(s.stdout, "7) Config init")
		fmt.Fprintln(s.stdout, "q) Quit")
		fmt.Fprintln(s.stdout)

		choice, err := s.prompt("Select: ")
		if err != nil {
			return err
		}
		switch choice {
		case "1":
			WriteDoctor(s.stdout, DetectHost())
			s.pause()
		case "2":
			if err := s.previewPlan(); err != nil {
				fmt.Fprintf(s.stderr, "error: %v\n", err)
			}
			s.pause()
		case "3":
			if err := s.changeProfile(); err != nil {
				fmt.Fprintf(s.stderr, "error: %v\n", err)
				s.pause()
			}
		case "4":
			value, err := s.prompt("Dotfiles URL (empty for embedded defaults): ")
			if err != nil {
				return err
			}
			s.dotfilesURL = value
		case "5":
			s.allowSourceBuilds = !s.allowSourceBuilds
		case "6":
			if err := s.install(); err != nil {
				fmt.Fprintf(s.stderr, "error: %v\n", err)
			}
			s.pause()
		case "7":
			path := filepath.Join(defaultHomeDir(), ".config", "ftsh", "config.json")
			if err := WriteDefaultConfig(path); err != nil {
				fmt.Fprintf(s.stderr, "error: %v\n", err)
			} else {
				fmt.Fprintf(s.stdout, "Wrote %s\n", path)
			}
			s.pause()
		case "q", "Q":
			return nil
		default:
			fmt.Fprintln(s.stderr, "unknown choice")
			s.pause()
		}
	}
}

func (s *tuiState) previewPlan() error {
	plan, err := BuildPlan(DetectHost(), s.planOptions())
	if err != nil {
		return err
	}
	WritePlan(s.stdout, plan)
	return nil
}

func (s *tuiState) changeProfile() error {
	profiles := BuiltinProfiles()
	fmt.Fprintln(s.stdout)
	for idx, profile := range profiles {
		fmt.Fprintf(s.stdout, "%d) %-10s %s\n", idx+1, profile.Name, profile.Description)
	}
	choice, err := s.prompt("Profile: ")
	if err != nil {
		return err
	}
	for idx, profile := range profiles {
		if choice == fmt.Sprint(idx+1) || choice == profile.Name {
			s.profileName = profile.Name
			return nil
		}
	}
	return fmt.Errorf("unknown profile %q", choice)
}

func (s *tuiState) install() error {
	plan, err := BuildPlan(DetectHost(), s.planOptions())
	if err != nil {
		return err
	}
	WritePlan(s.stdout, plan)
	fmt.Fprintln(s.stdout)
	confirm, err := s.prompt("Install this plan? Type yes to continue: ")
	if err != nil {
		return err
	}
	if confirm != "yes" {
		fmt.Fprintln(s.stdout, "Install cancelled.")
		return nil
	}
	installer := Installer{
		Stdout: s.stdout,
		Stderr: s.stderr,
		Stdin:  os.Stdin,
		Options: InstallOptions{
			AssumeYes:      false,
			NonInteractive: false,
			BackupExisting: true,
		},
	}
	return installer.Execute(plan)
}

func (s *tuiState) planOptions() PlanOptions {
	return PlanOptions{
		ProfileName:       s.profileName,
		DotfilesURL:       s.dotfilesURL,
		AllowSourceBuilds: s.allowSourceBuilds,
	}
}

func (s *tuiState) prompt(label string) (string, error) {
	fmt.Fprint(s.stdout, label)
	line, err := s.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (s *tuiState) pause() {
	fmt.Fprintln(s.stdout)
	_, _ = s.prompt("Press Enter to continue...")
}

func (s *tuiState) clear() {
	fmt.Fprint(s.stdout, "\033[H\033[2J")
}

func emptyAsNone(value string) string {
	if value == "" {
		return "embedded defaults"
	}
	return value
}
