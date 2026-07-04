package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gjkjk/zed/internal/config"
)

type openAI struct {
	cfg    *config.Config
	http   *http.Client
	apiURL string
}

func newOpenAI(cfg *config.Config) *openAI {
	url := cfg.BaseURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
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
	return &openAI{cfg: cfg, http: &http.Client{Transport: transport, Timeout: 10 * time.Minute}, apiURL: url}
}

func (o *openAI) Name() string { return "openai" }

type oaMsg struct {
	Role       string       `json:"role"`
	Content    string       `json:"content"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

type oaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaReq struct {
	Model     string   `json:"model"`
	Messages  []oaMsg  `json:"messages"`
	Tools     []oaTool `json:"tools,omitempty"`
	Stream    bool     `json:"stream"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

type oaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

func (o *openAI) buildBody(req Request) oaReq {
	body := oaReq{Model: req.Model, Stream: true, MaxTokens: req.MaxTokens}
	if req.System != "" {
		body.Messages = append(body.Messages, oaMsg{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		switch m.Role {
		case RoleUser:
			body.Messages = append(body.Messages, oaMsg{Role: "user", Content: m.Content})
		case RoleAssistant:
			msg := oaMsg{Role: "assistant", Content: m.Content}
			for _, tc := range m.ToolCalls {
				oc := oaToolCall{ID: tc.ID, Type: "function"}
				oc.Function.Name = tc.Name
				oc.Function.Arguments = orEmptyObj(tc.Args)
				msg.ToolCalls = append(msg.ToolCalls, oc)
			}
			body.Messages = append(body.Messages, msg)
		case RoleTool:
			body.Messages = append(body.Messages, oaMsg{
				Role: "tool", ToolCallID: m.ToolCallID, Content: m.Content,
			})
		}
	}
	for _, t := range req.Tools {
		var tool oaTool
		tool.Type = "function"
		tool.Function.Name = t.Name
		tool.Function.Description = t.Description
		tool.Function.Parameters = t.Parameters
		body.Tools = append(body.Tools, tool)
	}
	return body
}

func (o *openAI) Stream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	req.Model = orDefault(req.Model, o.cfg.Model)
	if req.MaxTokens == 0 {
		req.MaxTokens = o.cfg.MaxTokens
	}
	payload, err := json.Marshal(o.buildBody(req))
	if err != nil {
		return nil, err
	}
	// Read URL from cfg each time so /model switch takes effect immediately.
	apiURL := o.cfg.BaseURL
	if apiURL == "" {
		apiURL = o.apiURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// Read key from cfg each time so /model switch takes effect immediately.
	if o.cfg.AuthHeader != "" {
		// Provider-specific auth header (e.g. "api-key" for Xiaomi MiMo)
		httpReq.Header.Set(o.cfg.AuthHeader, o.cfg.APIKey)
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}

	resp, err := o.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return nil, newAPIError("openai", resp, buf.String())
	}

	out := make(chan StreamEvent)
	go o.readStream(resp, out)
	return out, nil
}

func (o *openAI) readStream(resp *http.Response, out chan<- StreamEvent) {
	defer resp.Body.Close()
	defer close(out)

	// A bufio.Reader (unlike bufio.Scanner) has no fixed max line length, so a
	// single large SSE event — e.g. a tool call whose arguments contain a whole
	// file — is never dropped mid-stream. That truncation was the source of the
	// "unexpected end of JSON input" errors when writing big files.
	reader := bufio.NewReaderSize(resp.Body, 256*1024)

	// accumulate streamed tool calls by index
	tools := map[int]*toolBuild{}
	order := []int{}
	truncated := false // model hit the output token limit

	for {
		// ReadString has no fixed length limit, so large SSE events are safe.
		raw, err := reader.ReadString('\n')
		line := strings.TrimRight(raw, "\r\n")
		if data, ok := strings.CutPrefix(line, "data:"); ok {
			data = strings.TrimSpace(data)
			if data == "[DONE]" {
				break
			}
			if data != "" {
				o.handleChunk(data, out, tools, &order, &truncated)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			out <- StreamEvent{Err: err}
			return
		}
	}

	// Flush tool calls in arrival order. Mark any as truncated so the agent can
	// surface a clear, actionable error instead of a cryptic JSON failure.
	for _, idx := range order {
		b := tools[idx]
		out <- StreamEvent{
			ToolCall:  &ToolCall{ID: b.id, Name: b.name, Args: b.args.String()},
			Truncated: truncated,
		}
	}
	out <- StreamEvent{Done: true, Truncated: truncated, Usage: &Usage{}}
}

// handleChunk parses one SSE data payload and updates stream state.
func (o *openAI) handleChunk(data string, out chan<- StreamEvent, tools map[int]*toolBuild, order *[]int, truncated *bool) {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content   string `json:"content"`
				Reasoning string `json:"reasoning"` // some models (GLM, DeepSeek-R1) use this instead of content
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return
	}
	for _, ch := range chunk.Choices {
		// Some reasoning models (GLM-5.2, DeepSeek-R1) send text in "reasoning" field.
		text := ch.Delta.Content
		if text == "" {
			text = ch.Delta.Reasoning
		}
		if text != "" {
			out <- StreamEvent{TextDelta: text}
		}
		for _, tc := range ch.Delta.ToolCalls {
			b, ok := tools[tc.Index]
			if !ok {
				b = &toolBuild{}
				tools[tc.Index] = b
				*order = append(*order, tc.Index)
			}
			if tc.ID != "" {
				b.id = tc.ID
			}
			if tc.Function.Name != "" {
				b.name = tc.Function.Name
			}
			b.args.WriteString(tc.Function.Arguments)
		}
		if ch.FinishReason == "length" {
			*truncated = true
		}
	}
}

// Complete sends a non-streaming request and returns the full response.
func (o *openAI) Complete(ctx context.Context, req Request) (Response, error) {
	req.Stream = false
	body := o.buildBody(req)
	body.Stream = false
	payload, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	apiURL := o.cfg.BaseURL
	if apiURL == "" {
		apiURL = o.apiURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.cfg.AuthHeader != "" {
		httpReq.Header.Set(o.cfg.AuthHeader, o.cfg.APIKey)
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}

	resp, err := o.http.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return Response{}, newAPIError("openai", resp, buf.String())
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				Reasoning string `json:"reasoning"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Response{}, err
	}

	var out Response
	if len(result.Choices) > 0 {
		out.Content = result.Choices[0].Message.Content
		if out.Content == "" {
			out.Content = result.Choices[0].Message.Reasoning
		}
		for _, tc := range result.Choices[0].Message.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: tc.Function.Arguments,
			})
		}
	}
	out.Usage = &Usage{
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
	}
	return out, nil
}
