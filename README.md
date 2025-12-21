# HostAtHome CLI

Command-line tool for managing game servers using Docker.

## Installation

### Debian/Ubuntu (.deb)

```bash
# Download the latest release
wget https://github.com/hostathome/cli/releases/download/v0.1.0/hostathome_0.1.0_amd64.deb

# Install
sudo dpkg -i hostathome_0.1.0_amd64.deb

# Verify installation
hostathome --version
```

For ARM64 systems (Raspberry Pi, etc.):
```bash
wget https://github.com/hostathome/cli/releases/download/v0.1.0/hostathome_0.1.0_arm64.deb
sudo dpkg -i hostathome_0.1.0_arm64.deb
```

## Quick Start

```bash
# Check if your system is ready
hostathome doctor

# List available games
hostathome list

# Install a game server
hostathome install minecraft

# Start the server
hostathome run minecraft

# Check status
hostathome status

# View logs
hostathome logs minecraft -f

# Stop the server
hostathome stop minecraft
```

### Updating

The CLI automatically checks for updates once per day and notifies you when a new version is available:

```bash
# Update to the latest version
hostathome update
```

### Cleanup Commands

```bash
# Remove container but keep data (can recreate with 'run')
hostathome remove minecraft

# Completely uninstall (removes container, image, and data)
hostathome uninstall minecraft
```

## Commands

| Command | Description |
|---------|-------------|
| `doctor` | Check system requirements (Docker, permissions, registry) |
| `list` | List available games in the registry |
| `install <game>` | Pull Docker image and create server directory |
| `run <game>` | Start the game server container |
| `stop <game>` | Stop the running container |
| `remove <game>` | Remove container but keep data directory |
| `uninstall <game>` | Remove container, image, and data (prompts for confirmation) |
| `status [game]` | Show status of running servers |
| `logs <game>` | View server logs (`-f` to follow, `-n` for line count) |
| `config <game>` | Edit server configuration in your default editor |
| `update` | Update CLI to the latest version from GitHub releases |

## Directory Structure

When you install a game, the CLI creates:

```
<game>-server/
├── save/           # World/game saves
├── mods/           # Plugins, addons
├── data/           # Runtime data
├── configs/
│   └── config.yaml # Server configuration
└── backup/         # User backups
```

## Configuration

Edit `./<game>-server/configs/config.yaml` to customize your server.

Example for Minecraft:
```yaml
server:
  motd: "My Minecraft Server"
  max-players: 20
  gamemode: survival
```

## Requirements

- Docker (installed and running)
- Linux, macOS, or Windows (with WSL2)
- User must be in the `docker` group (or use sudo)

Run `hostathome doctor` to verify your system is ready.

## Development

```bash
# Build binary
make build

# Install locally
make install

# Run tests
make test

# Build .deb package
make deb

# Build for all platforms
make release
```

## Uninstallation

To remove the CLI:
```bash
sudo dpkg -r hostathome
```

This removes the CLI but preserves any game server data you've created.
