 A tiny coding agent written in Go.

## Quick Start

1. Install Go (1.22+).
2. Install dependencies:

```bash
go mod tidy
```

3. Run:

```bash
go run ./cmd/tiny-agent
```

## Current Skeleton

- Starts a Bubble Tea CLI input interface.
- Type text and press `Enter` to echo input in the status line.
- Press `Ctrl+C` twice quickly to exit.
