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

```bash
# Edit server settings
hostathome config minecraft
# Prompts to restart after editing

# Edit mods
hostathome mods minecraft
# Prompts to restart after editing

# Or manually restart
hostathome restart minecraft
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
| `restart <game>` | Restart container to apply config/mod changes |
| `remove <game>` | Remove container but keep data directory |
| `uninstall <game>` | Remove container, image, and data (prompts for confirmation) |
| `status [game]` | Show status of running servers |
| `logs <game>` | View server logs (`-f` to follow, `-n` for line count) |
| `config <game>` | Edit server configuration (prompts to restart) |
| `mods <game>` | Edit mods configuration (prompts to restart) |
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

### Quick Development Workflow

```bash
# Build and install dev version
make build && sudo make install

# Or use this one-liner to uninstall current + install dev:
sudo dpkg -r hostathome 2>/dev/null; make build && sudo make install

# Test the dev version
hostathome --version
hostathome list
```

### Build Commands

```bash
# Build binary
make build

# Install locally (requires sudo)
make install

# Run tests
make test

# Build .deb package
make deb

# Build for all platforms
make release
```

### Development Tips

**Hot reload during development:**
```bash
# Use the binary directly without installing
make build
./bin/hostathome list

# Or create an alias
alias hah='./bin/hostathome'
hah list
```

**Testing changes:**
```bash
# 1. Make code changes
# 2. Rebuild
make build

# 3. Test without installing
./bin/hostathome install minecraft

# 4. When ready, install system-wide
sudo make install
```

## Development Mode

### Testing Local Server Images

When developing game server images, use the `--dev` flag to test without pulling from the registry:

```bash
# 1. Build your local image with :dev tag
cd hostathome/servers/minecraft-server
docker build -t minecraft-server:dev .

# 2. Run using local image
hostathome run minecraft --dev

# 3. Make changes to config, entrypoint, or Python scripts
# 4. Rebuild and restart
docker build -t minecraft-server:dev .
docker restart hostathome-minecraft

# Or use the Makefile for convenience
make rebuild
```

### The `--dev` Flag

- Uses local `<game>-server:dev` image instead of pulling from registry
- Useful for testing Docker image changes before pushing to production
- Skips the image pull step, so changes are tested immediately
- Requires the local image to exist first

### Example Workflow

```bash
# Start with building the dev image
cd hostathome/servers/minecraft-server
make build

# Start the server in dev mode
cd /path/to/servers
hostathome run minecraft --dev

# Test configuration changes
cat > minecraft-server/configs/mods.yaml <<EOF
loader: vanilla
modpack:
  platform: curseforge
  slug: "all-the-mods-9"
  api-key: "your-key"
mods: {}
EOF

# Quick rebuild and restart
cd hostathome/servers/minecraft-server
make rebuild

# Check logs
docker logs hostathome-minecraft -f
```

## Uninstallation

To remove the CLI:
```bash
sudo dpkg -r hostathome
```

This removes the CLI but preserves any game server data you've created.
