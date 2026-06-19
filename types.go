package main

import "time"

type HostInfo struct {
	Distro        DistroInfo         `json:"distro"`
	Package       PackageManagerInfo `json:"package_manager"`
	RunningAsRoot bool               `json:"running_as_root"`
	HasSudo       bool               `json:"has_sudo"`
	HasNetwork    bool               `json:"has_network"`
	Shell         string             `json:"shell"`
	HomeDir       string             `json:"home_dir"`
}

type DistroInfo struct {
	ID      string   `json:"id"`
	IDLike  []string `json:"id_like"`
	Name    string   `json:"name"`
	Version string   `json:"version"`
}

type PackageManagerInfo struct {
	Name      string `json:"name"`
	Detected  bool   `json:"detected"`
	Supported bool   `json:"supported"`
	Path      string `json:"path"`
}

type Profile struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Packages             []string `json:"packages"`
	ConfigProfile        string   `json:"config_profile"`
	AllowSourceBuilds    bool     `json:"allow_source_builds"`
	ChangeShellByDefault bool     `json:"change_shell_by_default"`
}

type PackageSpec struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Native      map[string]string `json:"native"`
	Source      string            `json:"source,omitempty"`
}

type PlanOptions struct {
	ProfileName       string `json:"profile_name"`
	DotfilesURL       string `json:"dotfiles_url,omitempty"`
	AllowSourceBuilds bool   `json:"allow_source_builds"`
}

type InstallPlan struct {
	GeneratedAt time.Time    `json:"generated_at"`
	Host        HostInfo     `json:"host"`
	Profile     Profile      `json:"profile"`
	Options     PlanOptions  `json:"options"`
	Actions     []PlanAction `json:"actions"`
	Warnings    []string     `json:"warnings"`
}

type PlanAction struct {
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Packages     []string `json:"packages,omitempty"`
	RequiresRoot bool     `json:"requires_root"`
}

type RunReport struct {
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt time.Time      `json:"finished_at"`
	Plan       InstallPlan    `json:"plan"`
	Results    []ActionResult `json:"results"`
}

type ActionResult struct {
	Action PlanAction `json:"action"`
	Status string     `json:"status"`
	Error  string     `json:"error,omitempty"`
}

type ConfigEntry struct {
	Source   string   `json:"source"`
	Target   string   `json:"target"`
	Mode     string   `json:"mode"`
	Profiles []string `json:"profiles"`
}
