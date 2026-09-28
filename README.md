# GyLiber Command Center

A local control-plane for the GyLiber analytical system.

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https%3A%2F%2Fgithub.com%2FEkantalokantarayatribharavahi%2Fgyliber-command-center)

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

The repository contains a Render Blueprint at `render.yaml`.

### Deploy with the button

Click **Deploy to Render** above, connect your GitHub account when prompted, select the repository, keep the Blueprint path as `render.yaml`, and click **Deploy Blueprint** / **Apply**. Render will create the configured free web service.

### Deploy from the Render Dashboard

1. Open the Render Dashboard.
2. Click **New → Blueprint**.
3. Connect `Ekantalokantarayatribharavahi/gyliber-command-center`.
4. Use branch `main` and Blueprint path `render.yaml`.
5. Click **Deploy Blueprint** / **Apply**.
6. Open the created web service and use its `.onrender.com` URL.

Render's current Blueprint workflow uses `render.yaml` at the repository root, and public repositories can be connected directly. citeturn719125search1turn719125search0
