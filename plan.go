package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

func BuildPlan(host HostInfo, opts PlanOptions) (InstallPlan, error) {
	if opts.ProfileName == "" {
		opts.ProfileName = "server"
	}
	profile, err := LookupProfile(opts.ProfileName)
	if err != nil {
		return InstallPlan{}, err
	}

	plan := InstallPlan{
		GeneratedAt: time.Now().UTC(),
		Host:        host,
		Profile:     profile,
		Options:     opts,
	}

	if !host.Package.Detected {
		plan.Warnings = append(plan.Warnings, "no package manager detected")
	} else if !host.Package.Supported {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("package manager %q is detected but not supported in v1", host.Package.Name))
	}
	if !host.RunningAsRoot && !host.HasSudo {
		plan.Warnings = append(plan.Warnings, "install requires root or sudo for package actions")
	}
	if !host.HasNetwork {
		plan.Warnings = append(plan.Warnings, "network connectivity check failed; package and dotfiles actions may fail")
	}

	nativePackages, sourceBuilds, packageWarnings := resolveProfilePackages(profile, host.Package.Name, opts.AllowSourceBuilds || profile.AllowSourceBuilds)
	plan.Warnings = append(plan.Warnings, packageWarnings...)
	if len(nativePackages) > 0 && host.Package.Supported {
		sort.Strings(nativePackages)
		plan.Actions = append(plan.Actions, PlanAction{
			Kind:         "install_packages",
			Name:         host.Package.Name,
			Description:  fmt.Sprintf("Install %d native packages with %s", len(nativePackages), host.Package.Name),
			Packages:     nativePackages,
			RequiresRoot: true,
		})
	}
	for _, name := range sourceBuilds {
		plan.Actions = append(plan.Actions, PlanAction{
			Kind:         "source_build",
			Name:         name,
			Description:  fmt.Sprintf("Build %s from source", name),
			RequiresRoot: false,
		})
	}

	configSource := "embedded defaults"
	if opts.DotfilesURL != "" {
		configSource = opts.DotfilesURL
	}
	plan.Actions = append(plan.Actions, PlanAction{
		Kind:        "apply_config",
		Name:        configSource,
		Description: fmt.Sprintf("Apply %s configuration from %s", profile.ConfigProfile, configSource),
	})

	if profile.ChangeShellByDefault {
		plan.Actions = append(plan.Actions, PlanAction{
			Kind:         "change_shell",
			Name:         "zsh",
			Description:  "Change the current user's login shell to zsh when zsh is available",
			RequiresRoot: false,
		})
	}

	return plan, nil
}

func resolveProfilePackages(profile Profile, manager string, allowSource bool) ([]string, []string, []string) {
	specs := PackageSpecs()
	native := []string{}
	sourceBuilds := []string{}
	warnings := []string{}
	for _, logical := range profile.Packages {
		spec, ok := specs[logical]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("profile %q references unknown package %q", profile.Name, logical))
			continue
		}
		if packageName, ok := spec.Native[manager]; ok && packageName != "" {
			native = append(native, packageName)
			continue
		}
		if spec.Source != "" && allowSource {
			sourceBuilds = append(sourceBuilds, logical)
			continue
		}
		if spec.Source != "" {
			warnings = append(warnings, fmt.Sprintf("%s has no native package for %s; rerun with --allow-source-builds to plan a source build", logical, manager))
		} else {
			warnings = append(warnings, fmt.Sprintf("%s has no package mapping for %s", logical, manager))
		}
	}
	return dedupe(native), dedupe(sourceBuilds), warnings
}

func WritePlan(w io.Writer, plan InstallPlan) {
	fmt.Fprintf(w, "Profile: %s\n", plan.Profile.Name)
	fmt.Fprintf(w, "Distro:  %s", emptyAsUnknown(plan.Host.Distro.ID))
	if plan.Host.Distro.Version != "" {
		fmt.Fprintf(w, " %s", plan.Host.Distro.Version)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Package: %s\n", emptyAsUnknown(plan.Host.Package.Name))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Actions:")
	for i, action := range plan.Actions {
		fmt.Fprintf(w, "  %d. %s", i+1, action.Description)
		if len(action.Packages) > 0 {
			fmt.Fprintf(w, " [%s]", strings.Join(action.Packages, ", "))
		}
		fmt.Fprintln(w)
	}
	if len(plan.Warnings) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Warnings:")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(w, "  - %s\n", warning)
		}
	}
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func emptyAsUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
