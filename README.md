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

### Modifying Configuration

Edit configuration files directly in your server directory:

```bash
# Edit server settings
nano minecraft-server/configs/config.yaml

# If you made changes, restart the server
hostathome restart minecraft

# View logs to confirm changes applied
hostathome logs minecraft -f
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
| `doctor` | Check system requirements (Docker, permissions, registry access) |
| `list` | List available games from the registry |
| `install <game>` | Pull Docker image and create server directory structure |
| `run <game>` | Start the game server container |
| `stop <game>` | Stop the running container |
| `restart <game>` | Restart container to apply config/mod changes |
| `remove <game>` | Remove container but keep data directory |
| `uninstall <game>` | Remove container, image, and data directory (prompts for confirmation) |
| `status [game]` | Show status of all or specific running servers in table format |
| `logs <game>` | View server logs (`-f` to follow, `-n <num>` for line count) |

**Note:** Configuration editing is done by directly modifying files in `<game>-server/configs/config.yaml` and `<game>-server/configs/mods.yaml` (if present).

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

## Architecture

### How It Works

HostAtHome CLI is a Docker management wrapper written in Go that simplifies game server operations:

1. **Registry System** - Fetches game definitions from a GitHub-hosted registry (YAML files)
2. **Docker Integration** - Uses Docker SDK to manage container lifecycle
3. **Configuration Mapping** - Containers convert `config.yaml` to game-specific formats via entrypoint scripts
4. **Volume Management** - All server data stored in standardized directory structure

### Components

**cmd/hostathome/main.go** - Command definitions and CLI routing using Cobra framework

**internal/docker/** - Docker SDK wrapper handling:
- Image pulling with timeout (5 minutes)
- Container creation with port mapping
- Container lifecycle (start, stop, restart, remove)
- Status queries and log retrieval
- Automatic volume/directory creation

**internal/registry/** - Registry management:
- Fetches game definitions from `https://raw.githubusercontent.com/hostathome/registry/main`
- Caches definitions locally for 1 hour
- Validates game names to prevent path traversal attacks
- Falls back to cache when offline

**internal/ui/** - Terminal formatting:
- Colored output with ASCII symbols
- Animated spinners for long operations
- Table formatting for status output
- Graceful degradation for piped output

**internal/config/** - Configuration management:
- Cache directory: `~/.hostathome/cache/registry/`
- Config directory: `~/.hostathome/`

### Data Flow

```
hostathome install minecraft
  ↓
Fetches minecraft.yaml from registry
  ↓
Creates minecraft-server/ with save/, configs/, mods/, backup/, data/ directories
  ↓
Pulls Docker image: ghcr.io/hostathome/minecraft-server
  ↓

hostathome run minecraft
  ↓
Creates Docker container with:
  - External ports: 30065 (players), 30066 (RCON)
  - Mounted volume: ./minecraft-server/ → /data
  - Labels for identification
  ↓
Container entrypoint:
  1. Symlinks /data/* to internal paths
  2. Reads config.yaml and converts to server.properties
  3. Starts Minecraft server
```

## Troubleshooting

### Docker Not Found or Permission Denied

**Error:** `permission denied while trying to connect to the Docker daemon`

**Solution:** Add your user to the docker group (requires docker to be installed):
```bash
sudo usermod -aG docker $USER
# Log out and log back in, or run:
newgrp docker
```

### Registry Connection Issues

**Error:** `failed to fetch registry` or `connection timeout`

**Solution:**
- Check internet connection: `ping github.com`
- The CLI caches game definitions locally for 1 hour - offline mode will use cached data
- Verify registry is accessible: `curl https://raw.githubusercontent.com/hostathome/registry/main/index.yaml`

### Port Already in Use

**Error:** `bind: address already in use`

**Solution:**
```bash
# Find what's using port 30065
sudo lsof -i :30065

# Stop the conflicting service or use different ports
# (Modify port mappings in the game server implementation)
```

### Container Crashes on Startup

**Solution:**
1. Check logs: `hostathome logs minecraft -f`
2. Verify config.yaml format is valid YAML
3. Ensure all required config keys are present
4. Check Docker image was pulled successfully: `docker images | grep minecraft`

### Stale Containers or Mount Issues

**Solution:**
```bash
# Remove container and data (careful!)
hostathome remove minecraft

# Recreate from scratch
hostathome run minecraft
```

### Check System Status

Always start troubleshooting with:
```bash
hostathome doctor
```

This checks Docker connectivity, permissions, and registry access.

