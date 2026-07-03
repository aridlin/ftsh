# FTSH

FTSH is a portable workstation bootstrapper for Linux systems. It turns a fresh SSH server, VPS, container, development machine, or recovery environment into a familiar shell/editor/tooling setup.

The current implementation is a Go CLI with built-in profiles, distro/package-manager detection, package planning, embedded fallback dotfiles, optional Git dotfiles, conflict backups, and install run reports.

## Build

Prebuilt binaries are available at <https://ftsh.aridlin.pl>.

```sh
go build -buildvcs=false -o ftsh .
```

`-buildvcs=false` is only needed in workspaces that are not real Git repositories.

## Usage

```sh
./ftsh doctor
./ftsh profiles
./ftsh plan --profile dev
./ftsh install --profile dev --yes
./ftsh install --profile personal --dotfiles git@example.com:me/dotfiles.git
./ftsh config init
./ftsh tui
```

`ftsh tui` opens a small dependency-free terminal menu for doctor checks, plan previews, profile selection, dotfiles URL entry, source-build toggling, install confirmation, and config initialization.

## Profiles

- `minimal`: rescue-friendly shell, Git, editor, and multiplexer basics.
- `server`: common interactive server tools.
- `dev`: development profile with Neovim, Zellij, search tools, and build tools.
- `recovery`: conservative setup that avoids login-shell changes.
- `personal`: full personal profile with dotfiles and source-build fallback enabled.

## Configuration

FTSH applies configuration from a manifest. If `--dotfiles` is provided, FTSH clones or updates the repo under `~/.cache/ftsh/dotfiles` and looks for one of:

- `manifest.json`
- `.ftsh/manifest.json`
- `ftsh-manifest.json`

Without `--dotfiles`, FTSH uses embedded fallback configs packaged into the binary.

Manifest entries look like this:

```json
{
  "source": "zshrc",
  "target": "~/.zshrc",
  "mode": "copy",
  "profiles": ["server", "dev", "personal"]
}
```

Existing target files are backed up by default under `~/.ftsh/backups/<timestamp>/`.

## Supported Systems

v1 supports:

- Debian/Ubuntu-family systems through `apt`
- Arch-family systems through `pacman`

Other package managers are detected and reported, but not installed through yet.
