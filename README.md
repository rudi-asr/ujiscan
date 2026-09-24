# ujiscan

Agentic Penetration Testing Platform built with Go and native HTML/JS.

## Features

- **Agentic Loop**: AI-driven feedback loop for dynamic attack chains
- **Native Go Backend**: No frameworks, pure `net/http`
- **Native Frontend**: HTML/CSS/JS without build tools
- **Tool Integration**: nmap, nuclei, and custom tool wrappers
- **Playbook Engine**: Sequential attack instructions
- **Real-time Feedback**: WebSocket for live scan updates

## Quick Start

### Build

```bash
go build -o ujiscan .
```

### Run

```bash
./ujiscan
```

Server starts at `http://localhost:8081`

## Project Structure

```
ujiscan/
├── cmd/server/       # HTTP server
├── internal/
│   ├── playbook/     # Playbook engine
│   ├── scanner/      # Scanner logic
│   ├── tools/        # Tool wrappers
│   ├── attack/       # Attack chains
│   └── models/       # Data models
├── web/              # Frontend (HTML/CSS/JS)
└── main.go           # Entry point
```

## Development

TODO: Add playbook format, tool catalog, agentic loop implementation.
