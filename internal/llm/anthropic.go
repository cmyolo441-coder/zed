package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gjkjk/zed/internal/config"
)

const anthropicVersion = "2023-06-01"

type anthropic struct {
	cfg    *config.Config
	http   *http.Client
	apiURL string
}

func newAnthropic(cfg *config.Config) *anthropic {
	url := cfg.BaseURL
	if url == "" {
		url = "https://api.anthropic.com/v1/messages"
	}
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   30 * time.Second,
		DisableKeepAlives:      false,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 5 * time.Minute,
	}
	return &anthropic{cfg: cfg, http: &http.Client{Transport: transport, Timeout: 10 * time.Minute}, apiURL: url}
}

func (a *anthropic) Name() string { return "anthropic" }

// --- wire format -------------------------------------------------------------

type antReq struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []antMsg     `json:"messages"`
	Tools     []antTool    `json:"tools,omitempty"`
	Stream    bool         `json:"stream"`
}

type antMsg struct {
	Role    string       `json:"role"`
	Content []antContent `json:"content"`
}

type antContent struct {
	Type string `json:"type"`
	// text
	Text string `json:"text,omitempty"`
	// tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type antTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

func (a *anthropic) buildBody(req Request) antReq {
	body := antReq{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		System:    req.System,
		Stream:    true,
	}
	for _, t := range req.Tools {
		body.Tools = append(body.Tools, antTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}
	for _, m := range req.Messages {
		switch m.Role {
		case RoleUser:
			body.Messages = append(body.Messages, antMsg{
				Role:    "user",
				Content: []antContent{{Type: "text", Text: m.Content}},
			})
		case RoleAssistant:
			var content []antContent
			if m.Content != "" {
				content = append(content, antContent{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				content = append(content, antContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: json.RawMessage(orEmptyObj(tc.Args)),
				})
			}
			body.Messages = append(body.Messages, antMsg{Role: "assistant", Content: content})
		case RoleTool:
			body.Messages = append(body.Messages, antMsg{
				Role: "user",
				Content: []antContent{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				}},
			})
		}
	}
	return body
}

func orEmptyObj(s string) string {
	if strings.TrimSpace(s) == "" {
		return "{}"
	}
	return s
}

// --- streaming ---------------------------------------------------------------

// Complete sends a non-streaming request and returns the full response. This
// powers swarm sub-agents (which don't need token-by-token streaming) and works
// on the Anthropic provider just like the OpenAI one.
func (a *anthropic) Complete(ctx context.Context, req Request) (Response, error) {
	req.Model = orDefault(req.Model, a.cfg.Model)
	if req.MaxTokens == 0 {
		req.MaxTokens = a.cfg.MaxTokens
	}
	body := a.buildBody(req)
	body.Stream = false
	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL, bytes.NewReader(payload))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return Response{}, newAPIError("anthropic", resp, buf.String())
	}

	var result struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Response{}, err
	}

	var out Response
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			out.Content += block.Text
		case "tool_use":
			args := string(block.Input)
			if strings.TrimSpace(args) == "" {
				args = "{}"
			}
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: args,
			})
		}
	}
	out.Usage = &Usage{
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
	}
	return out, nil
}

func (a *anthropic) Stream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	req.Model = orDefault(req.Model, a.cfg.Model)
	if req.MaxTokens == 0 {
		req.MaxTokens = a.cfg.MaxTokens
	}
	payload, err := json.Marshal(a.buildBody(req))
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return nil, newAPIError("anthropic", resp, buf.String())
	}

	out := make(chan StreamEvent)
	go a.readStream(resp, out)
	return out, nil
}

// currently-building tool call state during a stream
type toolBuild struct {
	id, name string
	args     strings.Builder
}

func (a *anthropic) readStream(resp *http.Response, out chan<- StreamEvent) {
	defer resp.Body.Close()
	defer close(out)

	// bufio.Reader has no fixed line-length cap, so large tool_use events never
	// get dropped mid-stream (the cause of truncated-JSON write failures).
	reader := bufio.NewReaderSize(resp.Body, 256*1024)

	var tb *toolBuild
	var usage Usage
	truncated := false

	for {
		raw, rerr := reader.ReadString('\n')
		line := strings.TrimRight(raw, "\r\n")
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			if rerr != nil {
				break
			}
			continue
		}
		data = strings.TrimSpace(data)
		if data == "" {
			if rerr != nil {
				break
			}
			continue
		}

		var evt struct {
			Type         string `json:"type"`
			Index        int    `json:"index"`
			ContentBlock struct {
				Type  string          `json:"type"`
				ID    string          `json:"id"`
				Name  string          `json:"name"`
				Text  string          `json:"text"`
				Input json.RawMessage `json:"input"`
			} `json:"content_block"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
				StopReason  string `json:"stop_reason"`
			} `json:"delta"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Message struct {
				Usage struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			if rerr != nil {
				break
			}
			continue
		}

		switch evt.Type {
		case "message_start":
			usage.InputTokens = evt.Message.Usage.InputTokens
		case "content_block_start":
			if evt.ContentBlock.Type == "tool_use" {
				tb = &toolBuild{id: evt.ContentBlock.ID, name: evt.ContentBlock.Name}
			}
		case "content_block_delta":
			switch evt.Delta.Type {
			case "text_delta":
				out <- StreamEvent{TextDelta: evt.Delta.Text}
			case "input_json_delta":
				if tb != nil {
					tb.args.WriteString(evt.Delta.PartialJSON)
				}
			}
		case "content_block_stop":
			if tb != nil {
				out <- StreamEvent{ToolCall: &ToolCall{
					ID: tb.id, Name: tb.name, Args: tb.args.String(),
				}, Truncated: truncated}
				tb = nil
			}
		case "message_delta":
			usage.OutputTokens = evt.Usage.OutputTokens
			if evt.Delta.StopReason == "max_tokens" {
				truncated = true
			}
		case "message_stop":
			out <- StreamEvent{Done: true, Truncated: truncated, Usage: &usage}
			return
		case "error":
			out <- StreamEvent{Err: fmt.Errorf("stream error: %s", data)}
			return
		}

		if rerr != nil {
			break
		}
	}
	out <- StreamEvent{Done: true, Truncated: truncated, Usage: &usage}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
