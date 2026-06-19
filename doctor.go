package main

import (
	"fmt"
	"io"
)

func WriteDoctor(w io.Writer, host HostInfo) {
	fmt.Fprintln(w, "FTSH doctor")
	fmt.Fprintf(w, "  Distro:          %s\n", emptyAsUnknown(host.Distro.ID))
	if len(host.Distro.IDLike) > 0 {
		fmt.Fprintf(w, "  Distro family:   %v\n", host.Distro.IDLike)
	}
	fmt.Fprintf(w, "  Package manager: %s\n", emptyAsUnknown(host.Package.Name))
	fmt.Fprintf(w, "  Supported:       %t\n", host.Package.Supported)
	fmt.Fprintf(w, "  Root:            %t\n", host.RunningAsRoot)
	fmt.Fprintf(w, "  Sudo:            %t\n", host.HasSudo)
	fmt.Fprintf(w, "  Network:         %t\n", host.HasNetwork)
	fmt.Fprintf(w, "  Shell:           %s\n", emptyAsUnknown(host.Shell))
	fmt.Fprintf(w, "  Home:            %s\n", host.HomeDir)
}
