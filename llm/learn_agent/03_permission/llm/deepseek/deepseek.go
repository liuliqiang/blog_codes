package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	agentloop "github.com/liuliqiang/llmagent/03_permission"
	"github.com/liuliqiang/log4go"
)

const defaultResponseBaseURL = "https://api.deepseek.com"

type deepseekClient struct {
	model           agentloop.Model
	responseBaseURL string
	apiKey          string
	httpClient      *http.Client
}

func NewDeepseekClient(options *deepseekClientOptions) agentloop.LLMClient {
	baseURL := options.responseBaseURL
	if baseURL == "" {
		baseURL = defaultResponseBaseURL
	}
	return &deepseekClient{
		model:           options.model,
		responseBaseURL: baseURL,
		apiKey:          options.apiKey,
		httpClient:      http.DefaultClient,
	}
}

// SendMessages calls the DeepSeek Responses API.
// See https://api-docs.deepseek.com/api/create-response
func (c *deepseekClient) SendMessages(
	ctx context.Context,
	model agentloop.Model,
	systemPrompt agentloop.Message,
	messages []agentloop.Message,
	tools []agentloop.Tool,
	opts agentloop.SendMessagesOpts,
) (agentloop.SendMessagesResponse, error) {
	log4go.DefaultLogger().Info(
		ctx,
		"SendMessages called with model: %s, systemPrompt: %+v, messages: %+v, tools: %+v, opts: %+v",
		model, systemPrompt, messages, tools, opts,
	)

	reqBody, err := buildRequest(ctx, model, systemPrompt, messages, tools)
	if err != nil {
		return agentloop.SendMessagesResponse{}, err
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "marshal request failed: %v, request: %+v", err, reqBody)
		return agentloop.SendMessagesResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	url := c.responseBaseURL + "/responses"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "new request failed: %v, url: %s", err, url)
		return agentloop.SendMessagesResponse{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "do request failed: %v, url: %s, payload: %s", err, url, payload)
		return agentloop.SendMessagesResponse{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "read response failed: %v, status: %d", err, resp.StatusCode)
		return agentloop.SendMessagesResponse{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		log4go.DefaultLogger().Error(ctx, "deepseek responses api failed: status %d, body: %s, payload: %s", resp.StatusCode, body, payload)
		return agentloop.SendMessagesResponse{}, fmt.Errorf("deepseek responses api: status %d: %s", resp.StatusCode, body)
	}

	var respBody responsesResponse
	if err := json.Unmarshal(body, &respBody); err != nil {
		log4go.DefaultLogger().Error(ctx, "unmarshal response failed: %v, body: %s", err, body)
		return agentloop.SendMessagesResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}
	return parseResponse(ctx, respBody)
}

func (c *deepseekClient) GetModel() agentloop.Model {
	return c.model
}

/* vvvvvvvvvvvvvvvvvvvvv request vvvvvvvvvvvvvvvvvvvvv */

type responsesRequest struct {
	Model        string        `json:"model"`
	Instructions string        `json:"instructions,omitempty"`
	Input        []interface{} `json:"input"`
	Tools        []toolDef     `json:"tools,omitempty"`
}

type toolDef struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type contentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type messageItem struct {
	Type    string      `json:"type"`
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []contentPart
}

type reasoningItem struct {
	Type    string        `json:"type"`
	Content []contentPart `json:"content"`
}

type functionCallItem struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type functionCallOutputItem struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

func buildRequest(
	ctx context.Context,
	model agentloop.Model,
	systemPrompt agentloop.Message,
	messages []agentloop.Message,
	tools []agentloop.Tool,
) (responsesRequest, error) {
	req := responsesRequest{Model: string(model)}

	if s, ok := systemPrompt.Content.(string); ok {
		req.Instructions = s
	}

	for _, msg := range messages {
		items, err := messageToInputItems(ctx, msg)
		if err != nil {
			return responsesRequest{}, err
		}
		req.Input = append(req.Input, items...)
	}

	for _, t := range tools {
		req.Tools = append(req.Tools, toolDef{
			Type:        "function",
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		})
	}
	return req, nil
}

func messageToInputItems(ctx context.Context, msg agentloop.Message) ([]interface{}, error) {
	switch content := msg.Content.(type) {
	case string:
		return []interface{}{messageItem{Type: "message", Role: string(msg.Role), Content: content}}, nil
	case []agentloop.MessagesBlock:
		return blocksToInputItems(ctx, msg.Role, content)
	default:
		log4go.DefaultLogger().Error(ctx, "unsupported message content type %T, message: %+v", msg.Content, msg)
		return nil, fmt.Errorf("unsupported message content type %T", msg.Content)
	}
}

func blocksToInputItems(ctx context.Context, role agentloop.MessageRole, blocks []agentloop.MessagesBlock) ([]interface{}, error) {
	textPartType := "input_text"
	if role == agentloop.MessageRoleAssistant {
		textPartType = "output_text"
	}

	var items []interface{}
	var textParts []contentPart
	flushText := func() {
		if len(textParts) > 0 {
			items = append(items, messageItem{Type: "message", Role: string(role), Content: textParts})
			textParts = nil
		}
	}

	for _, block := range blocks {
		switch block.Type {
		case agentloop.MessagesBlockTypeText:
			textParts = append(textParts, contentPart{Type: textPartType, Text: block.Text})
		case agentloop.MessagesBlockTypeReasoning:
			flushText()
			items = append(items, reasoningItem{
				Type:    "reasoning",
				Content: []contentPart{{Type: "reasoning_text", Text: block.Text}},
			})
		case agentloop.MessagesBlockTypeToolUse:
			flushText()
			args, err := json.Marshal(block.Input)
			if err != nil {
				log4go.DefaultLogger().Error(ctx, "marshal tool input failed: %v, block: %+v", err, block)
				return nil, fmt.Errorf("marshal tool input: %w", err)
			}
			items = append(items, functionCallItem{
				Type:      "function_call",
				CallID:    block.ID,
				Name:      block.Name,
				Arguments: string(args),
			})
		case agentloop.MessagesBlockTypeToolResult:
			flushText()
			items = append(items, functionCallOutputItem{
				Type:   "function_call_output",
				CallID: block.ToolUseID,
				Output: block.Content,
			})
		default:
			log4go.DefaultLogger().Error(ctx, "unsupported message block type %q, block: %+v", block.Type, block)
			return nil, fmt.Errorf("unsupported message block type %q", block.Type)
		}
	}
	flushText()
	return items, nil
}

/* ^^^^^^^^^^^^^^^^^^^^^ request ^^^^^^^^^^^^^^^^^^^^^ */

/* vvvvvvvvvvvvvvvvvvvvv response vvvvvvvvvvvvvvvvvvvv */

type responsesResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []outputItem `json:"output"`
}

type outputItem struct {
	Type    string        `json:"type"`
	ID      string        `json:"id"`
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`

	// for function_call only
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func parseResponse(ctx context.Context, resp responsesResponse) (agentloop.SendMessagesResponse, error) {
	if resp.Status == "failed" {
		msg := "unknown error"
		if resp.Error != nil {
			msg = fmt.Sprintf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		log4go.DefaultLogger().Error(ctx, "deepseek response %s failed: %s", resp.ID, msg)
		return agentloop.SendMessagesResponse{}, fmt.Errorf("deepseek response %s failed: %s", resp.ID, msg)
	}

	var blocks []agentloop.MessagesBlock
	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				blocks = append(blocks, agentloop.MessagesBlock{
					ID:   item.ID,
					Type: agentloop.MessagesBlockTypeText,
					Text: part.Text,
				})
			}
		case "reasoning":
			for _, part := range item.Content {
				blocks = append(blocks, agentloop.MessagesBlock{
					ID:   item.ID,
					Type: agentloop.MessagesBlockTypeReasoning,
					Text: part.Text,
				})
			}
		case "function_call":
			var input map[string]interface{}
			if item.Arguments != "" {
				if err := json.Unmarshal([]byte(item.Arguments), &input); err != nil {
					log4go.DefaultLogger().Error(ctx, "unmarshal function_call %s arguments failed: %v, arguments: %s", item.CallID, err, item.Arguments)
					return agentloop.SendMessagesResponse{}, fmt.Errorf("unmarshal function_call %s arguments: %w", item.CallID, err)
				}
			}
			blocks = append(blocks, agentloop.MessagesBlock{
				ID:    item.CallID,
				Type:  agentloop.MessagesBlockTypeToolUse,
				Name:  item.Name,
				Input: input,
			})
		default:
			log4go.DefaultLogger().Error(ctx, "unsupported output item type %q, item: %+v", item.Type, item)
			return agentloop.SendMessagesResponse{}, fmt.Errorf("unsupported output item type %q", item.Type)
		}
	}
	return agentloop.SendMessagesResponse{Content: blocks}, nil
}

/* ^^^^^^^^^^^^^^^^^^^^^ response ^^^^^^^^^^^^^^^^^^^^ */
