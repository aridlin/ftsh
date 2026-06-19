package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func BuildFromSource(name string) error {
	switch name {
	case "eza":
		return cargoInstall("eza")
	case "zellij":
		return cargoInstall("--locked", "zellij")
	case "fastfetch":
		return buildFastfetch()
	default:
		return fmt.Errorf("no source-build recipe for %s", name)
	}
}

func cargoInstall(args ...string) error {
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo is required for this source build: %w", err)
	}
	fullArgs := append([]string{"install"}, args...)
	return runInteractiveCommand("cargo", fullArgs...)
}

func buildFastfetch() error {
	for _, required := range []string{"git", "cmake"} {
		if _, err := exec.LookPath(required); err != nil {
			return fmt.Errorf("%s is required to build fastfetch: %w", required, err)
		}
	}

	src := filepath.Join(defaultHomeDir(), ".cache", "ftsh", "src", "fastfetch")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
			return err
		}
		if err := runInteractiveCommand("git", "clone", "https://github.com/fastfetch-cli/fastfetch.git", src); err != nil {
			return err
		}
	} else {
		if err := runInteractiveCommand("git", "-C", src, "pull", "--ff-only"); err != nil {
			return err
		}
	}

	buildDir := filepath.Join(src, "build")
	if err := runInteractiveCommand("cmake", "-S", src, "-B", buildDir, "-DCMAKE_BUILD_TYPE=Release"); err != nil {
		return err
	}
	if err := runInteractiveCommand("cmake", "--build", buildDir); err != nil {
		return err
	}

	binDir := filepath.Join(defaultHomeDir(), ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	return copyExecutable(filepath.Join(buildDir, "fastfetch"), filepath.Join(binDir, "fastfetch"))
}

func copyExecutable(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o755)
}

func runInteractiveCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
