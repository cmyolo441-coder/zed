package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gjkjk/zed/internal/mission"
)

type MissionControl struct{ WorkDir string }
func (t *MissionControl) Name() string { return "mission_control" }
func (t *MissionControl) Description() string { return "Persistent multi-goal mission control: start, list, pause, resume, complete, cancel, journal, artifact. Stores .zed-missions.json. Args: {action,id,goal,note,artifact,json}" }
func (t *MissionControl) RequiresApproval() bool { return true }
func (t *MissionControl) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"action":map[string]any{"type":"string"},"id":map[string]any{"type":"integer"},"goal":map[string]any{"type":"string"},"note":map[string]any{"type":"string"},"artifact":map[string]any{"type":"string"},"json":map[string]any{"type":"boolean"}},"required":[]string{"action"}} }
func (t *MissionControl) Execute(_ context.Context, args string) (string,error){
	var a struct{Action string `json:"action"`; ID int `json:"id"`; Goal string `json:"goal"`; Note string `json:"note"`; Artifact string `json:"artifact"`; JSON bool `json:"json"`}
	if strings.TrimSpace(args)==""{a.Action="list"}else if err:=parseArgs(args,&a);err!=nil{return "",err}
	var out any
	var err error
	switch a.Action{
	case "start": out,err=mission.Start(t.WorkDir,a.Goal)
	case "pause": out,err=mission.SetStatus(t.WorkDir,a.ID,mission.StatusPaused,a.Note)
	case "resume": out,err=mission.SetStatus(t.WorkDir,a.ID,mission.StatusActive,a.Note)
	case "complete": out,err=mission.SetStatus(t.WorkDir,a.ID,mission.StatusCompleted,a.Note)
	case "cancel": out,err=mission.SetStatus(t.WorkDir,a.ID,mission.StatusCancelled,a.Note)
	case "journal": out,err=mission.AddJournal(t.WorkDir,a.ID,a.Note)
	case "artifact": out,err=mission.AddArtifact(t.WorkDir,a.ID,a.Artifact)
	case "list","": s,e:=mission.List(t.WorkDir); if e!=nil{return "",e}; if a.JSON{out=s}else{return mission.Render(s),nil}
	default: return "",fmt.Errorf("unknown mission action %q",a.Action)
	}
	if err!=nil{return "",err}
	b,_:=json.MarshalIndent(out,"","  ")
	return string(b),nil
}

func ParseMissionID(s string) int { fields:=strings.Fields(s); if len(fields)==0{return 0}; n,_:=strconv.Atoi(fields[0]); return n }
