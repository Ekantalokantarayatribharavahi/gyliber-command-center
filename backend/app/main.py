from __future__ import annotations

import json
import uuid
from datetime import datetime, timezone

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles
from pathlib import Path
from pydantic import BaseModel

from .adapters import ADAPTERS
from .db import connect

app = FastAPI(title="GyLiber Command Center", version="0.1.0")
app.add_middleware(CORSMiddleware, allow_origins=["*"], allow_methods=["*"], allow_headers=["*"])

def now() -> str:
    return datetime.now(timezone.utc).isoformat()

class RunRequest(BaseModel):
    division: str
    operation: str
    seed: int | None = None
    dataset_hash: str | None = None

@app.get("/", include_in_schema=False)
def index():
    if (FRONTEND_DIST / "index.html").exists():
        return FileResponse(FRONTEND_DIST / "index.html")
    return {"service":"GyLiber Command Center","status":"backend-ready"}

@app.on_event("startup")
def startup() -> None:
    connect().close()

@app.get("/api/system/status")
def system_status():
    return {"system":"OPERATIONAL","session":"MVP-01","active_game":"LOTTO","rule_version":"lotto-current","dataset_state":"UNVERIFIED","analysis_state":"READY","audit_state":"READY","mission_time":now()}

@app.get("/api/system/health")
def system_health():
    return {"sqlite":"READY","artifact_store":"READY","orchestrator":"READY","api":"READY","adapters":[a.status().__dict__ for a in ADAPTERS]}

@app.get("/api/runs")
def list_runs():
    with connect() as db:
        rows=db.execute("SELECT * FROM runs ORDER BY created_at DESC").fetchall()
    return [dict(r) for r in rows]

@app.get("/api/runs/{run_id}")
def get_run(run_id:str):
    with connect() as db:
        run=db.execute("SELECT * FROM runs WHERE id=?",(run_id,)).fetchone()
        events=db.execute("SELECT * FROM run_events WHERE run_id=? ORDER BY created_at",(run_id,)).fetchall()
        artifacts=db.execute("SELECT * FROM artifacts WHERE run_id=? ORDER BY created_at",(run_id,)).fetchall()
    return {"run":dict(run) if run else None,"events":[dict(x) for x in events],"artifacts":[dict(x) for x in artifacts]}

@app.post("/api/runs")
def create_run(req:RunRequest):
    run_id=f"RUN-{datetime.now().strftime('%Y%m%d-%H%M%S')}-{uuid.uuid4().hex[:6]}"
    created=now()
    with connect() as db:
        db.execute("INSERT INTO runs VALUES (?,?,?,?,?,?,?,?)",(run_id,req.division,req.operation,"RUNNING",req.seed,req.dataset_hash,created,None))
        db.execute("INSERT INTO run_events(run_id,event_type,message,created_at) VALUES(?,?,?,?)",(run_id,"RUN_STARTED",f"{req.division}: {req.operation}",created))
        db.commit()
    return {"id":run_id,"status":"RUNNING"}

@app.post("/api/runs/{run_id}/complete")
def complete_run(run_id:str):
    ended=now()
    with connect() as db:
        db.execute("UPDATE runs SET status='COMPLETE',completed_at=? WHERE id=?",(ended,run_id))
        db.execute("INSERT INTO run_events(run_id,event_type,message,created_at) VALUES(?,?,?,?)",(run_id,"RUN_COMPLETED","Run completed.",ended))
        db.commit()
    return {"id":run_id,"status":"COMPLETE"}

@app.get("/api/artifacts")
def list_artifacts():
    with connect() as db:
        rows=db.execute("SELECT * FROM artifacts ORDER BY created_at DESC").fetchall()
    return [dict(r) for r in rows]

@app.get("/api/lineage")
def lineage():
    return {"nodes":[
      {"id":"source","label":"OFFICIAL SOURCE","state":"READY"},
      {"id":"raw","label":"RAW CAPTURE","state":"READY"},
      {"id":"dataset","label":"VALIDATED DATASET","state":"READY"},
      {"id":"forensics","label":"FORENSICS RESULT","state":"READY"},
      {"id":"crowding","label":"CROWDING SCORE","state":"READY"},
      {"id":"candidate","label":"CANDIDATE RANK","state":"READY"},
      {"id":"audit","label":"FINAL AUDIT","state":"READY"}],
      "edges":[["source","raw"],["raw","dataset"],["dataset","forensics"],["dataset","candidate"],["candidate","crowding"],["crowding","audit"],["candidate","audit"]]}
