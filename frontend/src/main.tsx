import React,{useEffect,useState} from "react";
import {createRoot} from "react-dom/client";
import "./styles.css";

const API="http://localhost:8000";
type Status=Record<string,any>;
type Run=Record<string,any>;

function App(){
  const [status,setStatus]=useState<Status>({});
  const [runs,setRuns]=useState<Run[]>([]);
  const [selected,setSelected]=useState("COMMAND");

  const load=async()=>{const [s,r]=await Promise.all([fetch(API+"/api/system/status").then(x=>x.json()),fetch(API+"/api/runs").then(x=>x.json())]);setStatus(s);setRuns(r)};
  useEffect(()=>{load();const id=setInterval(load,3000);return()=>clearInterval(id)},[]);

  const launch=async(division:string,operation:string)=>{
    await fetch(API+"/api/runs",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({division,operation,seed:42})});
    load();
  };

  const nav=["COMMAND","EVIDENCE VAULT","FORENSICS LAB","CROWDING LAB","COMBINATION LAB","AUDIT ROOM","DATA ARCHIVE","RUN HISTORY","SYSTEM HEALTH"];
  const panels=[
    ["EVIDENCE VAULT","214 sources","VERIFIED","Inspect provenance"],
    ["FORENSICS LAB","8 hypotheses","READY","Review evidence states"],
    ["CROWDING LAB","10,000 candidates","MODEL 0.1.0","Inspect feature wall"],
    ["COMBINATION LAB","Candidate population","READY","Open pipeline"]
  ];

  return <div className="app">
    <header className="bridge"><div><span className="eyebrow">GYLIBER</span><h1>COMMAND CENTER</h1></div><div className="bridgeMeta"><b>SYSTEM {status.system||"CONNECTING"}</b><span>SESSION {status.session||"—"}</span><span>{status.active_game||"LOTTO"} · {status.rule_version||"—"}</span><span>{new Date(status.mission_time||Date.now()).toLocaleString()}</span></div></header>
    <nav className="nav">{nav.map(x=><button className={selected===x?"active":""} onClick={()=>setSelected(x)}>{x}</button>)}</nav>
    <main>
      <section className="hero"><div><span className="kicker">OPERATIONS FLOOR</span><h2>{selected}</h2><p>Observe → Verify → Analyse → Model → Generate → Audit</p></div><div className="clock">{new Date(status.mission_time||Date.now()).toISOString().replace("T"," ").slice(0,19)} <small>UTC</small></div></section>
      {selected==="COMMAND" && <>
        <section className="grid4">{panels.map(([title,value,state,action])=><article className="panel"><span className="label">{title}</span><strong>{value}</strong><span className="state">{state}</span><button onClick={()=>setSelected(title)}>{action} →</button></article>)}</section>
        <section className="split"><article className="panel"><div className="sectionTitle">OPERATIONS FLOW</div><div className="flow">{["OFFICIAL EVIDENCE","VALIDATED DATA","FORENSICS","CROWDING","COMBINATION","INDEPENDENT AUDIT"].map((x,i)=><div className="flowNode"><span>{String(i+1).padStart(2,"0")}</span>{x}</div>)}</div></article><article className="panel"><div className="sectionTitle">RUN CONTROL</div><p className="muted">Approved operations become auditable runs.</p><div className="actions"><button onClick={()=>launch("Evidence","Verify source")}>VERIFY EVIDENCE</button><button onClick={()=>launch("Forensics","Run analysis")}>RUN FORENSICS</button><button onClick={()=>launch("Combination","Generate candidates")}>GENERATE</button></div></article></section>
      </>}
      {selected==="RUN HISTORY" && <section className="panel"><div className="sectionTitle">RUN HISTORY</div>{runs.length===0?<p className="muted">No runs yet.</p>:<table><thead><tr><th>TIME</th><th>DIVISION</th><th>OPERATION</th><th>SEED</th><th>STATUS</th></tr></thead><tbody>{runs.map(r=><tr><td>{r.created_at}</td><td>{r.division}</td><td>{r.operation}</td><td>{r.seed??"—"}</td><td className="state">{r.status}</td></tr>)}</tbody></table>}</section>}
      {selected!=="COMMAND"&&selected!=="RUN HISTORY"&&<section className="split"><article className="panel"><div className="sectionTitle">STATUS</div><div className="bigState">READY</div><p className="muted">Control surface established for {selected.toLowerCase()}.</p></article><article className="panel"><div className="sectionTitle">LINEAGE</div><div className="lineage">SOURCE → DATA → FORENSICS → CROWDING → COMBINATION → AUDIT</div></article></section>}
    </main>
    <footer>GYLIBER COMMAND CENTER · CONTROL PLANE · PURCHASE ACTION NOT EXECUTED</footer>
  </div>
}
createRoot(document.getElementById("root")!).render(<App/>);
