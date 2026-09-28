# GyLiber Command Center

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/Ekantalokantarayatribharavahi/gyliber-command-center)

A local control-plane for the GyLiber analytical system.

The MVP provides a dark command-center web UI, a REST API, a SQLite run/event/artifact registry, adapter interfaces for the four analytical instruments, lineage metadata, and reproducible run records.

## Stack

- React + TypeScript + Vite frontend
- FastAPI backend
- SQLite persistence
- Adapter boundary for Data Forge, Forensics, Crowding, and Combination Lab

## Run

Backend:

```bash
cd backend
python -m pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Open the Vite URL shown by the frontend.

## Design

The repositories remain independent scientific instruments. The Command Center is the orchestration and observability layer above them.
