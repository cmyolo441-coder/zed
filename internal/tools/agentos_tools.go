package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gjkjk/zed/internal/agentos"
)

type AgentOSTool struct{ WorkDir, ToolName string }
func (t *AgentOSTool) Name() string { return t.ToolName }
func (t *AgentOSTool) Description() string { return "BITTU CHAUHAN Agent OS capability: " + t.ToolName }
func (t *AgentOSTool) RequiresApproval() bool { switch t.ToolName { case "ui_composer","macro_recorder","workflow_engine","code_movie","autonomous_app_builder","blueprint_engine","theme_generator","persona_engine","knowledge_pack_builder","learning_system","tutorial_builder","readme_designer": return true }; return false }
func (t *AgentOSTool) Schema() map[string]any { return map[string]any{"type":"object","properties":map[string]any{"action":map[string]any{"type":"string"},"path":map[string]any{"type":"string"},"name":map[string]any{"type":"string"},"prompt":map[string]any{"type":"string"},"goal":map[string]any{"type":"string"},"text":map[string]any{"type":"string"},"json":map[string]any{"type":"boolean"},"run":map[string]any{"type":"boolean"},"spec":map[string]any{"type":"object"},"skill":map[string]any{"type":"object"},"vars":map[string]any{"type":"object"}}} }
func (t *AgentOSTool) Execute(_ context.Context, args string) (string,error){
	var a struct{ Action string `json:"action"`; Path string `json:"path"`; Name string `json:"name"`; Prompt string `json:"prompt"`; Goal string `json:"goal"`; Text string `json:"text"`; JSON bool `json:"json"`; Run bool `json:"run"`; Spec json.RawMessage `json:"spec"`; Skill agentos.Skill `json:"skill"`; Vars map[string]string `json:"vars"` }
	if strings.TrimSpace(args)!="" && strings.TrimSpace(args)!="{}" { if err:=parseArgs(args,&a); err!=nil{return "",err} }
	var r agentos.Result; var err error
	switch t.ToolName {
	case "enterprise_vision_ui_builder": r,err=agentos.VisionUIBuilder(t.WorkDir,a.Path)
	case "prompt_os": r,err=agentos.PromptOS(t.WorkDir,a.Action,a.Name,a.Vars)
	case "agent_role_studio": r,err=agentos.RoleStudio(t.WorkDir,first(a.Goal,a.Text,a.Prompt))
	case "skill_marketplace": r,err=agentos.SkillMarketplace(t.WorkDir,a.Action,a.Skill)
	case "ui_composer": var spec agentos.UISpec; _=json.Unmarshal(a.Spec,&spec); r,err=agentos.UIComposer(t.WorkDir,spec)
	case "macro_recorder": r,err=agentos.MacroRecorder(t.WorkDir,a.Action,a.Name,first(a.Text,a.Prompt))
	case "workflow_engine": r,err=agentos.WorkflowEngine(t.WorkDir,a.Action,a.Path,a.Run)
	case "agent_desktop": r,err=agentos.DesktopView(t.WorkDir)
	case "code_movie": r,err=agentos.CodeMovie(t.WorkDir,first(a.Name,a.Goal,a.Prompt))
	case "terminal_animation_engine": r,err=agentos.AnimationEngine(first(a.Name,a.Prompt,"pulse"))
	case "autonomous_app_builder": r,err=agentos.AppBuilder(t.WorkDir,first(a.Goal,a.Prompt,a.Text,"terminal app"))
	case "blueprint_engine": r,err=agentos.BlueprintEngine(t.WorkDir,first(a.Text,a.Prompt))
	case "theme_generator": r,err=agentos.ThemeGenerator(t.WorkDir,first(a.Prompt,a.Name,"bittu green"))
	case "persona_engine": r,err=agentos.PersonaEngine(t.WorkDir,first(a.Name,a.Prompt))
	case "knowledge_pack_builder": r,err=agentos.KnowledgePackBuilder(t.WorkDir)
	case "learning_system": r,err=agentos.LearningSystem(t.WorkDir,a.Action,first(a.Text,a.Prompt))
	case "tutorial_builder": r,err=agentos.TutorialBuilder(t.WorkDir)
	case "readme_designer": r,err=agentos.ReadmeDesigner(t.WorkDir)
	case "voice_command_mode": r,err=agentos.VoiceCommandMode(first(a.Text,a.Prompt))
	case "self_upgrade_kernel": r,err=agentos.SelfUpgradeKernel(t.WorkDir)
	default: return "", fmt.Errorf("unknown agent os tool %s", t.ToolName)
	}
	if err!=nil{return "",err}; return r.Text(),nil
}
func first(vals ...string) string { for _,v:=range vals{ if v!=""{return v} }; return "" }

func NewAgentOSTools(workDir string) []Tool { names:=[]string{"enterprise_vision_ui_builder","prompt_os","agent_role_studio","skill_marketplace","ui_composer","macro_recorder","workflow_engine","agent_desktop","code_movie","terminal_animation_engine","autonomous_app_builder","blueprint_engine","theme_generator","persona_engine","knowledge_pack_builder","learning_system","tutorial_builder","readme_designer","voice_command_mode","self_upgrade_kernel"}; out:=make([]Tool,0,len(names)); for _,n:=range names{out=append(out,&AgentOSTool{WorkDir:workDir,ToolName:n})}; return out }
