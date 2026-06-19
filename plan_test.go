package main

import "testing"

func TestBuildPlanForAptDevWarnsForMissingNativeWithoutSourceBuilds(t *testing.T) {
	host := HostInfo{
		Distro:        DistroInfo{ID: "ubuntu"},
		Package:       PackageManagerInfo{Name: "apt", Detected: true, Supported: true},
		RunningAsRoot: true,
		HasNetwork:    true,
	}

	plan, err := BuildPlan(host, PlanOptions{ProfileName: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Profile.Name != "dev" {
		t.Fatalf("profile = %q, want dev", plan.Profile.Name)
	}
	if len(plan.Actions) == 0 || plan.Actions[0].Kind != "install_packages" {
		t.Fatalf("first action = %#v, want install_packages", plan.Actions)
	}
	if !contains(plan.Actions[0].Packages, "fd-find") {
		t.Fatalf("apt packages = %#v, want fd-find mapping", plan.Actions[0].Packages)
	}
	if len(plan.Warnings) == 0 {
		t.Fatal("expected warning for zellij missing native apt package")
	}
	if hasActionKind(plan.Actions, "source_build") {
		t.Fatal("did not expect source_build action without opt-in")
	}
}

func TestBuildPlanAllowsSourceBuildActions(t *testing.T) {
	host := HostInfo{
		Distro:        DistroInfo{ID: "ubuntu"},
		Package:       PackageManagerInfo{Name: "apt", Detected: true, Supported: true},
		RunningAsRoot: true,
		HasNetwork:    true,
	}

	plan, err := BuildPlan(host, PlanOptions{ProfileName: "dev", AllowSourceBuilds: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasActionKind(plan.Actions, "source_build") {
		t.Fatalf("actions = %#v, want source_build", plan.Actions)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasActionKind(actions []PlanAction, kind string) bool {
	for _, action := range actions {
		if action.Kind == kind {
			return true
		}
	}
	return false
}
