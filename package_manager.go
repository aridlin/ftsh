package main

import (
	"fmt"
	"os"
	"os/exec"
)

type PackageManager interface {
	Name() string
	IsInstalled(pkg string) bool
	Install(pkgs []string, assumeYes bool) error
}

func NewPackageManager(info PackageManagerInfo, root bool, sudo bool) (PackageManager, error) {
	runner := commandRunner{root: root, sudo: sudo}
	switch info.Name {
	case "apt":
		return aptManager{runner: runner}, nil
	case "pacman":
		return pacmanManager{runner: runner}, nil
	default:
		return nil, fmt.Errorf("unsupported package manager %q", info.Name)
	}
}

type commandRunner struct {
	root bool
	sudo bool
}

func (r commandRunner) Run(name string, args ...string) error {
	if !r.root {
		if !r.sudo {
			return fmt.Errorf("command %s requires root or sudo", name)
		}
		args = append([]string{name}, args...)
		name = "sudo"
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

type aptManager struct {
	runner commandRunner
}

func (m aptManager) Name() string { return "apt" }

func (m aptManager) IsInstalled(pkg string) bool {
	return exec.Command("dpkg", "-s", pkg).Run() == nil
}

func (m aptManager) Install(pkgs []string, assumeYes bool) error {
	if len(pkgs) == 0 {
		return nil
	}
	if err := m.runner.Run("apt-get", "update"); err != nil {
		return err
	}
	args := []string{"install"}
	if assumeYes {
		args = append(args, "-y")
	}
	args = append(args, pkgs...)
	return m.runner.Run("apt-get", args...)
}

type pacmanManager struct {
	runner commandRunner
}

func (m pacmanManager) Name() string { return "pacman" }

func (m pacmanManager) IsInstalled(pkg string) bool {
	return exec.Command("pacman", "-Qi", pkg).Run() == nil
}

func (m pacmanManager) Install(pkgs []string, assumeYes bool) error {
	if len(pkgs) == 0 {
		return nil
	}
	args := []string{"-Sy", "--needed"}
	if assumeYes {
		args = append(args, "--noconfirm")
	}
	args = append(args, pkgs...)
	return m.runner.Run("pacman", args...)
}
