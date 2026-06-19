package main

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"
)

func DetectHost() HostInfo {
	distro := DetectDistro("/etc/os-release")
	pm := DetectPackageManager(distro)
	return HostInfo{
		Distro:        distro,
		Package:       pm,
		RunningAsRoot: os.Geteuid() == 0,
		HasSudo:       commandExists("sudo"),
		HasNetwork:    hasLikelyNetwork(),
		Shell:         os.Getenv("SHELL"),
		HomeDir:       defaultHomeDir(),
	}
}

func DetectDistro(path string) DistroInfo {
	values := map[string]string{}
	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			key, value, ok := strings.Cut(scanner.Text(), "=")
			if !ok {
				continue
			}
			values[key] = strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	return DistroInfo{
		ID:      strings.ToLower(values["ID"]),
		IDLike:  splitIDLike(values["ID_LIKE"]),
		Name:    values["NAME"],
		Version: values["VERSION_ID"],
	}
}

func DetectPackageManager(distro DistroInfo) PackageManagerInfo {
	candidates := preferredPackageManagers(distro)
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return PackageManagerInfo{Name: name, Detected: true, Supported: name == "apt" || name == "pacman", Path: path}
		}
	}
	for _, name := range []string{"apt", "pacman", "dnf", "zypper", "apk", "xbps-install"} {
		if path, err := exec.LookPath(name); err == nil {
			return PackageManagerInfo{Name: name, Detected: true, Supported: name == "apt" || name == "pacman", Path: path}
		}
	}
	return PackageManagerInfo{}
}

func preferredPackageManagers(d DistroInfo) []string {
	ids := append([]string{d.ID}, d.IDLike...)
	for _, id := range ids {
		switch id {
		case "debian", "ubuntu":
			return []string{"apt"}
		case "arch", "archlinux":
			return []string{"pacman"}
		}
	}
	return nil
}

func splitIDLike(value string) []string {
	if value == "" {
		return nil
	}
	fields := strings.Fields(strings.ToLower(value))
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		out = append(out, strings.Trim(field, `"`))
	}
	return out
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func defaultHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return "."
}

func hasLikelyNetwork() bool {
	conn, err := net.DialTimeout("tcp", "1.1.1.1:53", 700*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
