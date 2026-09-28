package main

import (
  "crypto/rand"
  "database/sql"
  "encoding/hex"
  "encoding/json"
  "errors"
  "fmt"
  "log"
  "net/http"
  "os"
  "path/filepath"
  "strings"
  "time"

  _ "modernc.org/sqlite"
)

const appVersion = "0.2.0"
const schema = \`
CREATE TABLE IF NOT EXISTS runs (
 id TEXT PRIMARY KEY, division TEXT NOT NULL, operation TEXT NOT NULL,
 status TEXT NOT NULL, seed INTEGER, dataset_hash TEXT,
 created_at TEXT NOT NULL, completed_at TEXT
);
CREATE TABLE IF NOT EXISTS run_events (
 id INTEGER PRIMARY KEY AUTOINCREMENT, run_id TEXT NOT NULL,
 event_type TEXT NOT NULL, message TEXT NOT NULL, created_at TEXT NOT NULL,
 FOREIGN KEY(run_id) REFERENCES runs(id)
);
CREATE TABLE IF NOT EXISTS artifacts (
 id TEXT PRIMARY KEY, run_id TEXT NOT NULL, name TEXT NOT NULL,
 producer TEXT NOT NULL, integrity_status TEXT NOT NULL, dataset_hash TEXT,
 created_at TEXT NOT NULL, FOREIGN KEY(run_id) REFERENCES runs(id)
);
CREATE TABLE IF NOT EXISTS system_changes (
 id TEXT PRIMARY KEY, component TEXT NOT NULL, previous_version TEXT,
 new_version TEXT NOT NULL, reason TEXT NOT NULL, created_at TEXT NOT NULL
);\`

type Server struct{ db *sql.DB; frontend string }
type RunRequest struct {
 Division string \`json:"division"\`
 Operation string \`json:"operation"\`
 Seed *int64 \`json:"seed,omitempty"\`
 DatasetHash string \`json:"dataset_hash,omitempty"\`
}
type AdapterStatus struct {
 Name string \`json:"name"\`
 State string \`json:"state"\`
 OperationCount int \`json:"operation_count"\`
 Version string \`json:"version"\`
}
type Adapter struct{ Name string }
func (a Adapter) Status() AdapterStatus { return AdapterStatus{a.Name,"READY",0,"0.1.0"} }

var adapters = []Adapter{
 {Name:"EVIDENCE VAULT"}, {Name:"FORENSICS LAB"},
 {Name:"CROWDING LAB"}, {Name:"COMBINATION LAB"},
}

func main() {
 dbPath:=getenv("DB_PATH","gyliber_command_center.db")
 frontend:=getenv("FRONTEND_DIST","../frontend/dist")
 db,err:=sql.Open("sqlite",dbPath);if err!=nil{log.Fatal(err)};defer db.Close()
 if _,err=db.Exec(schema);err!=nil{log.Fatal(err)}
 if _,err=db.Exec("PRAGMA foreign_keys = ON");err!=nil{log.Fatal(err)}
 db.SetMaxOpenConns(1);db.SetMaxIdleConns(1)
 s:=&Server{db:db,frontend:frontend}
 mux:=http.NewServeMux()
 mux.HandleFunc("/api/system/status",s.status)
 mux.HandleFunc("/api/system/health",s.health)
 mux.HandleFunc("/api/runs",s.runs)
 mux.HandleFunc("/api/runs/",s.runByID)
 mux.HandleFunc("/api/artifacts",s.artifacts)
 mux.HandleFunc("/api/lineage",s.lineage)
 mux.HandleFunc("/",s.frontendHandler)
 server:=&http.Server{
   Addr:":"+getenv("PORT","8080"),Handler:securityHeaders(logging(mux)),
   ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,
   WriteTimeout:30*time.Second,IdleTimeout:60*time.Second,MaxHeaderBytes:1<<20,
 }
 log.Printf("GyLiber Command Center %s listening on %s",appVersion,server.Addr)
 log.Fatal(server.ListenAndServe())
}

func (s *Server) status(w http.ResponseWriter,r *http.Request){
 if !method(w,r,http.MethodGet){return}
 writeJSON(w,200,map[string]any{
  "system":"OPERATIONAL","session":"MVP-01","active_game":"LOTTO",
  "rule_version":"lotto-current","dataset_state":"UNVERIFIED",
  "analysis_state":"READY","audit_state":"READY","version":appVersion,
  "mission_time":now(),
 })
}

func (s *Server) health(w http.ResponseWriter,r *http.Request){
 if !method(w,r,http.MethodGet){return}
 state:="READY";if err:=s.db.PingContext(r.Context());err!=nil{state="FAILED"}
 statuses:=make([]AdapterStatus,0,len(adapters))
 for _,a:=range adapters{statuses=append(statuses,a.Status())}
 writeJSON(w,200,map[string]any{
  "sqlite":state,"artifact_store":"READY","orchestrator":"READY","api":"READY","adapters":statuses,
 })
}

func (s *Server) runs(w http.ResponseWriter,r *http.Request){
 switch r.Method{
 case http.MethodGet:
  rows,err:=s.db.QueryContext(r.Context(),"SELECT id,division,operation,status,seed,dataset_hash,created_at,completed_at FROM runs ORDER BY created_at DESC")
  if err!=nil{http.Error(w,"failed to read runs",500);return};defer rows.Close()
  out:=make([]map[string]any,0)
  for rows.Next(){
   var id,div,op,status,created string;var seed sql.NullInt64;var hash,completed sql.NullString
   if err:=rows.Scan(&id,&div,&op,&status,&seed,&hash,&created,&completed);err!=nil{http.Error(w,"failed to decode runs",500);return}
   out=append(out,map[string]any{"id":id,"division":div,"operation":op,"status":status,"seed":nullableInt(seed),"dataset_hash":nullableString(hash),"created_at":created,"completed_at":nullableString(completed)})
  }
  if err:=rows.Err();err!=nil{http.Error(w,"failed to read runs",500);return};writeJSON(w,200,out)
 case http.MethodPost:
  var req RunRequest;if err:=decodeJSON(w,r,&req,4096);err!=nil{http.Error(w,err.Error(),400);return}
  if strings.TrimSpace(req.Division)==""||strings.TrimSpace(req.Operation)==""{http.Error(w,"division and operation are required",400);return}
  id,err:=newID("RUN");if err!=nil{http.Error(w,"failed to create run ID",500);return};created:=now()
  _,err=s.db.ExecContext(r.Context(),"INSERT INTO runs(id,division,operation,status,seed,dataset_hash,created_at,completed_at) VALUES(?,?,?,?,?,?,?,NULL)",id,req.Division,req.Operation,"RUNNING",req.Seed,emptyToNil(req.DatasetHash),created)
  if err!=nil{http.Error(w,"failed to create run",500);return}
  _,_=s.db.ExecContext(r.Context(),"INSERT INTO run_events(run_id,event_type,message,created_at) VALUES(?,?,?,?)",id,"RUN_STARTED",req.Division+": "+req.Operation,created)
  writeJSON(w,http.StatusCreated,map[string]any{"id":id,"status":"RUNNING"})
 default: http.Error(w,"method not allowed",405)
 }
}

func (s *Server) runByID(w http.ResponseWriter,r *http.Request){
 parts:=strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path,"/api/runs/"),"/"),"/")
 if len(parts)==1&&r.Method==http.MethodGet{s.getRun(w,r,parts[0]);return}
 if len(parts)==2&&parts[1]=="complete"&&r.Method==http.MethodPost{s.completeRun(w,r,parts[0]);return}
 http.Error(w,"not found",404)
}

func (s *Server) getRun(w http.ResponseWriter,r *http.Request,id string){
 var div,op,status,created string;var seed sql.NullInt64;var hash,completed sql.NullString
 err:=s.db.QueryRowContext(r.Context(),"SELECT division,operation,status,seed,dataset_hash,created_at,completed_at FROM runs WHERE id=?",id).Scan(&div,&op,&status,&seed,&hash,&created,&completed)
 if errors.Is(err,sql.ErrNoRows){writeJSON(w,404,map[string]string{"error":"run not found"});return};if err!=nil{http.Error(w,"failed to read run",500);return}
 rows,err:=s.db.QueryContext(r.Context(),"SELECT event_type,message,created_at FROM run_events WHERE run_id=? ORDER BY created_at,id",id)
 if err!=nil{http.Error(w,"failed to read events",500);return};defer rows.Close()
 events:=make([]map[string]string,0)
 for rows.Next(){var t,m,c string;if err:=rows.Scan(&t,&m,&c);err!=nil{http.Error(w,"failed to decode events",500);return};events=append(events,map[string]string{"event_type":t,"message":m,"created_at":c})}
 arts,err:=s.db.QueryContext(r.Context(),"SELECT id,name,producer,integrity_status,dataset_hash,created_at FROM artifacts WHERE run_id=? ORDER BY created_at,id",id)
 if err!=nil{http.Error(w,"failed to read artifacts",500);return};defer arts.Close()
 artifacts:=make([]map[string]any,0)
 for arts.Next(){var aid,name,producer,integrity,createdAt string;var ahash sql.NullString;if err:=arts.Scan(&aid,&name,&producer,&integrity,&ahash,&createdAt);err!=nil{http.Error(w,"failed to decode artifacts",500);return};artifacts=append(artifacts,map[string]any{"id":aid,"name":name,"producer":producer,"integrity_status":integrity,"dataset_hash":nullableString(ahash),"created_at":createdAt})}
 writeJSON(w,200,map[string]any{
  "run":map[string]any{"id":id,"division":div,"operation":op,"status":status,"seed":nullableInt(seed),"dataset_hash":nullableString(hash),"created_at":created,"completed_at":nullableString(completed)},
  "events":events,"artifacts":artifacts,
 })
}

func (s *Server) completeRun(w http.ResponseWriter,r *http.Request,id string){
 ended:=now();res,err:=s.db.ExecContext(r.Context(),"UPDATE runs SET status='COMPLETE',completed_at=? WHERE id=?",ended,id)
 if err!=nil{http.Error(w,"failed to complete run",500);return};n,_:=res.RowsAffected();if n==0{writeJSON(w,404,map[string]string{"error":"run not found"});return}
 _,_=s.db.ExecContext(r.Context(),"INSERT INTO run_events(run_id,event_type,message,created_at) VALUES(?,?,?,?)",id,"RUN_COMPLETED","Run completed.",ended)
 writeJSON(w,200,map[string]string{"id":id,"status":"COMPLETE"})
}

func (s *Server) artifacts(w http.ResponseWriter,r *http.Request){
 if !method(w,r,http.MethodGet){return}
 rows,err:=s.db.QueryContext(r.Context(),"SELECT id,run_id,name,producer,integrity_status,dataset_hash,created_at FROM artifacts ORDER BY created_at DESC")
 if err!=nil{http.Error(w,"failed to read artifacts",500);return};defer rows.Close()
 out:=make([]map[string]any,0)
 for rows.Next(){var id,runID,name,producer,integrity,created string;var hash sql.NullString;if err:=rows.Scan(&id,&runID,&name,&producer,&integrity,&hash,&created);err!=nil{http.Error(w,"failed to decode artifacts",500);return};out=append(out,map[string]any{"id":id,"run_id":runID,"name":name,"producer":producer,"integrity_status":integrity,"dataset_hash":nullableString(hash),"created_at":created})}
 writeJSON(w,200,out)
}

func (s *Server) lineage(w http.ResponseWriter,r *http.Request){
 if !method(w,r,http.MethodGet){return}
 writeJSON(w,200,map[string]any{
  "nodes":[]map[string]string{
   {"id":"source","label":"OFFICIAL SOURCE","state":"READY"},{"id":"raw","label":"RAW CAPTURE","state":"READY"},
   {"id":"dataset","label":"VALIDATED DATASET","state":"READY"},{"id":"forensics","label":"FORENSICS RESULT","state":"READY"},
   {"id":"crowding","label":"CROWDING SCORE","state":"READY"},{"id":"candidate","label":"CANDIDATE RANK","state":"READY"},
   {"id":"audit","label":"FINAL AUDIT","state":"READY"},
  },
  "edges":[][]string{{"source","raw"},{"raw","dataset"},{"dataset","forensics"},{"dataset","candidate"},{"candidate","crowding"},{"crowding","audit"},{"candidate","audit"}},
 })
}

func (s *Server) frontendHandler(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodGet&&r.Method!=http.MethodHead{http.Error(w,"method not allowed",405);return}
 clean:=filepath.Clean(strings.TrimPrefix(r.URL.Path,"/"));if clean=="."||clean==""{clean="index.html"}
 path:=filepath.Join(s.frontend,clean)
 if info,err:=os.Stat(path);err==nil&&!info.IsDir(){http.ServeFile(w,r,path);return}
 index:=filepath.Join(s.frontend,"index.html");if _,err:=os.Stat(index);err==nil{http.ServeFile(w,r,index);return}
 writeJSON(w,200,map[string]string{"service":"GyLiber Command Center","status":"backend-ready","version":appVersion})
}

func decodeJSON(w http.ResponseWriter,r *http.Request,dst any,limit int64)error{
 r.Body=http.MaxBytesReader(w,r.Body,limit);dec:=json.NewDecoder(r.Body);dec.DisallowUnknownFields();if err:=dec.Decode(dst);err!=nil{return fmt.Errorf("invalid JSON: %w",err)};return nil
}
func writeJSON(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.WriteHeader(status);_=json.NewEncoder(w).Encode(v)}
func method(w http.ResponseWriter,r *http.Request,expected string)bool{if r.Method!=expected{http.Error(w,"method not allowed",405);return false};return true}
func securityHeaders(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
 w.Header().Set("X-Content-Type-Options","nosniff");w.Header().Set("X-Frame-Options","DENY");w.Header().Set("Referrer-Policy","no-referrer")
 w.Header().Set("Permissions-Policy","geolocation=(), microphone=(), camera=()")
 w.Header().Set("Content-Security-Policy","default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; font-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
 next.ServeHTTP(w,r)
})}
func logging(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){start:=time.Now();next.ServeHTTP(w,r);log.Printf("%s %s %s",r.Method,r.URL.Path,time.Since(start))})}
func newID(prefix string)(string,error){b:=make([]byte,6);if _,err:=rand.Read(b);err!=nil{return "",err};return prefix+"-"+time.Now().UTC().Format("20060102-150405")+"-"+hex.EncodeToString(b),nil}
func now()string{return time.Now().UTC().Format(time.RFC3339Nano)}
func getenv(k,f string)string{if v:=strings.TrimSpace(os.Getenv(k));v!=""{return v};return f}
func emptyToNil(v string)any{if strings.TrimSpace(v)==""{return nil};return v}
func nullableString(v sql.NullString)any{if v.Valid{return v.String};return nil}
func nullableInt(v sql.NullInt64)any{if v.Valid{return v.Int64};return nil}
