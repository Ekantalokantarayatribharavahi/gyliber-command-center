# GyLiber Command Center

A local control-plane for the GyLiber analytical system.

## Stack

- React + TypeScript + Vite frontend
- Go orchestration/API backend
- SQLite persistence
- Adapter boundary for Data Forge, Forensics, Crowding, and Combination Lab

The analytical repositories remain independently executable. The Command Center is their control and observability layer.

## Run locally

Build the frontend:

```bash
cd frontend
npm install
npm run build
cd ..
```

Run the Go control plane:

```bash
go mod download
FRONTEND_DIST=frontend/dist go run ./backend
```

## Deployment

The repository contains a Render Blueprint using the Go native runtime. Render supports `runtime: go`; sync the Blueprint after committing the runtime change.
