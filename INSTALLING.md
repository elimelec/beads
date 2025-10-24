# Installing bd

Complete installation guide for all platforms.

## Quick Install (Recommended)

### Homebrew (macOS/Linux)

```bash
brew tap steveyegge/beads
brew install bd
```

**Why Homebrew?**
- ✅ Simple one-command install
- ✅ Automatic updates via `brew upgrade`
- ✅ No need to install Go
- ✅ Handles PATH setup automatically

### Quick Install Script (All Platforms)

```bash
curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
```

The installer will:
- Detect your platform (macOS/Linux, amd64/arm64)
- Install via `go install` if Go is available
- Fall back to building from source if needed
- Guide you through PATH setup if necessary

## Platform-Specific Installation

### macOS

**Via Homebrew** (recommended):
```bash
brew tap steveyegge/beads
brew install bd
```

**Via go install**:
```bash
go install github.com/steveyegge/beads/cmd/bd@latest
```

**From source**:
```bash
git clone https://github.com/steveyegge/beads
cd beads
go build -o bd ./cmd/bd
sudo mv bd /usr/local/bin/
```

### Linux

**Via Homebrew** (works on Linux too):
```bash
brew tap steveyegge/beads
brew install bd
```

**Arch Linux** (AUR):
```bash
# Install from AUR
yay -S beads-git
# or
paru -S beads-git
```

Thanks to [@v4rgas](https://github.com/v4rgas) for maintaining the AUR package!

**Via go install**:
```bash
go install github.com/steveyegge/beads/cmd/bd@latest
```

**From source**:
```bash
git clone https://github.com/steveyegge/beads
cd beads
go build -o bd ./cmd/bd
sudo mv bd /usr/local/bin/
```

### Windows 11

Beads now ships with native Windows support—no MSYS or MinGW required.

**Prerequisites:**
- [Go 1.24+](https://go.dev/dl/) installed (add `%USERPROFILE%\go\bin` to your `PATH`)
- Git for Windows

**Via PowerShell script**:
```pwsh
irm https://raw.githubusercontent.com/steveyegge/beads/main/install.ps1 | iex
```

**Via go install**:
```pwsh
go install github.com/steveyegge/beads/cmd/bd@latest
```

**From source**:
```pwsh
git clone https://github.com/steveyegge/beads
cd beads
go build -o bd.exe ./cmd/bd
Move-Item bd.exe $env:USERPROFILE\AppData\Local\Microsoft\WindowsApps\
```

**Verify installation**:
```pwsh
bd version
```

**Windows notes:**
- The background daemon listens on a loopback TCP endpoint recorded in `.beads\bd.sock`
- Keep that metadata file intact
- Allow `bd.exe` loopback traffic through any host firewall

## IDE and Editor Integrations

### Claude Code Plugin

For Claude Code users, the beads plugin provides slash commands and MCP tools.

**Prerequisites:**
1. First, install the bd CLI (see above)
2. Then install the plugin:

```bash
# In Claude Code
/plugin marketplace add steveyegge/beads
/plugin install beads
# Restart Claude Code
```

The plugin includes:
- Slash commands: `/bd-ready`, `/bd-create`, `/bd-show`, `/bd-update`, `/bd-close`, etc.
- Full MCP server with all bd tools
- Task agent for autonomous execution

See [PLUGIN.md](PLUGIN.md) for complete plugin documentation.

### MCP Server (For Sourcegraph Amp, Claude Desktop, and other MCP clients)

If you're using an MCP-compatible tool other than Claude Code:

```bash
# Using uv (recommended)
uv tool install beads-mcp

# Or using pip
pip install beads-mcp
```

**Configuration for Claude Desktop** (macOS):

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "beads": {
      "command": "beads-mcp"
    }
  }
}
```

**Configuration for Sourcegraph Amp**:

Add to your MCP settings:

```json
{
  "beads": {
    "command": "beads-mcp",
    "args": []
  }
}
```

**What you get:**
- Full bd functionality exposed via MCP protocol
- Tools for creating, updating, listing, and closing issues
- Ready work detection and dependency management
- All without requiring Bash commands

See [integrations/beads-mcp/README.md](integrations/beads-mcp/README.md) for detailed MCP server documentation.

## Verifying Installation

After installing, verify bd is working:

```bash
bd version
bd help
```

## Troubleshooting Installation

### `bd: command not found`

bd is not in your PATH. Either:

```bash
# Check if installed
go list -f {{.Target}} github.com/steveyegge/beads/cmd/bd

# Add Go bin to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$PATH:$(go env GOPATH)/bin"

# Or reinstall
go install github.com/steveyegge/beads/cmd/bd@latest
```

### `zsh: killed bd` or crashes on macOS

Some users report crashes when running `bd init` or other commands on macOS. This is typically caused by CGO/SQLite compatibility issues.

**Workaround:**
```bash
# Build with CGO enabled
CGO_ENABLED=1 go install github.com/steveyegge/beads/cmd/bd@latest

# Or if building from source
git clone https://github.com/steveyegge/beads
cd beads
CGO_ENABLED=1 go build -o bd ./cmd/bd
sudo mv bd /usr/local/bin/
```

If you installed via Homebrew, this shouldn't be necessary as the formula already enables CGO. If you're still seeing crashes with the Homebrew version, please [file an issue](https://github.com/steveyegge/beads/issues).

## Next Steps

After installation:

1. **Initialize a project**: `cd your-project && bd init`
2. **Configure your agent**: Add bd instructions to `AGENTS.md` (see [README.md](README.md#quick-start))
3. **Learn the basics**: Run `bd quickstart` for an interactive tutorial
4. **Explore examples**: Check out the [examples/](examples/) directory

## Updating bd

### Homebrew

```bash
brew upgrade bd
```

### go install

```bash
go install github.com/steveyegge/beads/cmd/bd@latest
```

### From source

```bash
cd beads
git pull
go build -o bd ./cmd/bd
sudo mv bd /usr/local/bin/
```
