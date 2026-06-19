package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectDistroParsesOSRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "os-release")
	err := os.WriteFile(path, []byte(`NAME="Ubuntu"
ID=ubuntu
VERSION_ID="24.04"
ID_LIKE="debian"
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	distro := DetectDistro(path)
	if distro.ID != "ubuntu" {
		t.Fatalf("ID = %q, want ubuntu", distro.ID)
	}
	if distro.Version != "24.04" {
		t.Fatalf("Version = %q, want 24.04", distro.Version)
	}
	if len(distro.IDLike) != 1 || distro.IDLike[0] != "debian" {
		t.Fatalf("IDLike = %#v, want debian", distro.IDLike)
	}
}

func TestPreferredPackageManagers(t *testing.T) {
	tests := []struct {
		name string
		in   DistroInfo
		want string
	}{
		{name: "ubuntu", in: DistroInfo{ID: "ubuntu"}, want: "apt"},
		{name: "debian-like", in: DistroInfo{ID: "pop", IDLike: []string{"ubuntu", "debian"}}, want: "apt"},
		{name: "arch", in: DistroInfo{ID: "arch"}, want: "pacman"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preferredPackageManagers(tt.in)
			if len(got) == 0 || got[0] != tt.want {
				t.Fatalf("preferredPackageManagers() = %#v, want first %q", got, tt.want)
			}
		})
	}
}
