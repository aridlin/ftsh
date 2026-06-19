package main

import "fmt"

func BuiltinProfiles() []Profile {
	return []Profile{
		{
			Name:          "minimal",
			Description:   "Small rescue-friendly baseline with shell, git, and editor essentials.",
			Packages:      []string{"zsh", "git", "curl", "wget", "neovim", "tmux"},
			ConfigProfile: "minimal",
		},
		{
			Name:          "server",
			Description:   "Interactive server baseline with common search, file, and admin tools.",
			Packages:      []string{"zsh", "git", "curl", "wget", "neovim", "tmux", "ripgrep", "fd", "eza", "fastfetch"},
			ConfigProfile: "server",
		},
		{
			Name:                 "dev",
			Description:          "Development workstation profile with shell, editor, multiplexer, and build tools.",
			Packages:             []string{"zsh", "git", "curl", "wget", "neovim", "zellij", "ripgrep", "fd", "eza", "fastfetch", "build-essential"},
			ConfigProfile:        "dev",
			ChangeShellByDefault: true,
		},
		{
			Name:          "recovery",
			Description:   "Conservative recovery profile that avoids login-shell changes by default.",
			Packages:      []string{"git", "curl", "wget", "vim", "tmux", "ripgrep"},
			ConfigProfile: "minimal",
		},
		{
			Name:                 "personal",
			Description:          "Full personal environment with dotfiles, zsh, Neovim, and Zellij.",
			Packages:             []string{"zsh", "git", "curl", "wget", "neovim", "zellij", "ripgrep", "fd", "eza", "fastfetch", "build-essential", "zsh-autosuggestions", "zsh-syntax-highlighting"},
			ConfigProfile:        "personal",
			ChangeShellByDefault: true,
		},
	}
}

func LookupProfile(name string) (Profile, error) {
	for _, profile := range BuiltinProfiles() {
		if profile.Name == name {
			return profile, nil
		}
	}
	return Profile{}, fmt.Errorf("unknown profile %q", name)
}

func PackageSpecs() map[string]PackageSpec {
	return map[string]PackageSpec{
		"zsh": {
			Name: "zsh", Description: "Z shell",
			Native: map[string]string{"apt": "zsh", "pacman": "zsh"},
		},
		"git": {
			Name: "git", Description: "Git version control",
			Native: map[string]string{"apt": "git", "pacman": "git"},
		},
		"curl": {
			Name: "curl", Description: "HTTP client",
			Native: map[string]string{"apt": "curl", "pacman": "curl"},
		},
		"wget": {
			Name: "wget", Description: "HTTP download tool",
			Native: map[string]string{"apt": "wget", "pacman": "wget"},
		},
		"vim": {
			Name: "vim", Description: "Vim editor",
			Native: map[string]string{"apt": "vim", "pacman": "vim"},
		},
		"neovim": {
			Name: "neovim", Description: "Neovim editor",
			Native: map[string]string{"apt": "neovim", "pacman": "neovim"},
		},
		"tmux": {
			Name: "tmux", Description: "terminal multiplexer",
			Native: map[string]string{"apt": "tmux", "pacman": "tmux"},
		},
		"zellij": {
			Name: "zellij", Description: "terminal workspace",
			Native: map[string]string{"pacman": "zellij"},
			Source: "cargo",
		},
		"ripgrep": {
			Name: "ripgrep", Description: "fast recursive search",
			Native: map[string]string{"apt": "ripgrep", "pacman": "ripgrep"},
		},
		"fd": {
			Name: "fd", Description: "fast file finder",
			Native: map[string]string{"apt": "fd-find", "pacman": "fd"},
		},
		"eza": {
			Name: "eza", Description: "modern ls replacement",
			Native: map[string]string{"pacman": "eza"},
			Source: "cargo",
		},
		"fastfetch": {
			Name: "fastfetch", Description: "system information summary",
			Native: map[string]string{"pacman": "fastfetch"},
			Source: "cmake",
		},
		"build-essential": {
			Name: "build-essential", Description: "compiler and build toolchain",
			Native: map[string]string{"apt": "build-essential", "pacman": "base-devel"},
		},
		"zsh-autosuggestions": {
			Name: "zsh-autosuggestions", Description: "zsh command autosuggestions",
			Native: map[string]string{"apt": "zsh-autosuggestions", "pacman": "zsh-autosuggestions"},
		},
		"zsh-syntax-highlighting": {
			Name: "zsh-syntax-highlighting", Description: "zsh command syntax highlighting",
			Native: map[string]string{"apt": "zsh-syntax-highlighting", "pacman": "zsh-syntax-highlighting"},
		},
	}
}
