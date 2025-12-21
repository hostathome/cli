# HostAtHome CLI

Command-line tool for managing game servers using Docker.

## Installation

### From Source
```bash
cd cli
make build
make install
```

### Binary Release
Download from [Releases](https://github.com/hostathome/cli/releases) and add to PATH.

## Usage

```bash
# List available games
hostathome list

# Install a game server
hostathome install minecraft

# Start the server
hostathome run minecraft

# Check status
hostathome status

# Stop the server
hostathome stop minecraft
```

## Commands

| Command | Description |
|---------|-------------|
| `list` | List available games in the registry |
| `install <game>` | Pull Docker image and create server directory |
| `run <game>` | Start the game server container |
| `stop <game>` | Stop the running container |
| `status [game]` | Show status of running servers |

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

- Docker (running)
- Linux, macOS, or Windows (with WSL2)

## Development

```bash
# Build
make build

# Run tests
make test

# Sync registry from ../registry
make sync-registry

# Build release binaries
make release
```
