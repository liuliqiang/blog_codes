package deepseek

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	agentloop "github.com/liuliqiang/llmagent/08_context_compact"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) agentloop.LLMClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	opts := NewDeepseekClientOptions(agentloop.ModelDeepseekFlash)
	opts.WithResponseBaseURL(server.URL).WithAPIKey("test-key")
	return NewDeepseekClient(opts)
}

func TestSendMessages_BuildsRequest(t *testing.T) {
	var gotPath, gotAuth, gotContentType string
	var gotBody map[string]interface{}

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Fatalf("invalid request json: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","status":"completed","output":[]}`))
	})

	messages := []agentloop.Message{
		{Role: agentloop.MessageRoleUser, Content: "list files"},
		{Role: agentloop.MessageRoleAssistant, Content: []agentloop.MessagesBlock{
			{Type: agentloop.MessagesBlockTypeReasoning, Text: "need to run ls"},
			{Type: agentloop.MessagesBlockTypeText, Text: "Let me check."},
			{ID: "call_1", Type: agentloop.MessagesBlockTypeToolUse, Name: "bash", Input: map[string]interface{}{"command": "ls"}},
		}},
		{Role: agentloop.MessageRoleUser, Content: []agentloop.MessagesBlock{
			{Type: agentloop.MessagesBlockTypeToolResult, ToolUseID: "call_1", Content: "a.txt"},
		}},
	}
	tools := []agentloop.Tool{{
		Name:        "bash",
		Description: "run a bash command",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{"command": map[string]interface{}{"type": "string"}},
		},
	}}

	_, err := client.SendMessages(
		context.Background(),
		agentloop.ModelDeepseekFlash,
		agentloop.Message{Role: agentloop.MessageRoleSystem, Content: "You are a senior software engineer."},
		messages,
		tools,
		nil,
	)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	if gotPath != "/responses" {
		t.Errorf("path = %q, want /responses", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}

	want := map[string]interface{}{
		"model":        "deepseek-flash",
		"instructions": "You are a senior software engineer.",
		"input": []interface{}{
			map[string]interface{}{"type": "message", "role": "user", "content": "list files"},
			map[string]interface{}{"type": "reasoning", "content": []interface{}{
				map[string]interface{}{"type": "reasoning_text", "text": "need to run ls"},
			}},
			map[string]interface{}{"type": "message", "role": "assistant", "content": []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Let me check."},
			}},
			map[string]interface{}{"type": "function_call", "call_id": "call_1", "name": "bash", "arguments": `{"command":"ls"}`},
			map[string]interface{}{"type": "function_call_output", "call_id": "call_1", "output": "a.txt"},
		},
		"tools": []interface{}{
			map[string]interface{}{
				"type":        "function",
				"name":        "bash",
				"description": "run a bash command",
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{"command": map[string]interface{}{"type": "string"}},
				},
			},
		},
	}
	if !reflect.DeepEqual(gotBody, want) {
		gotJSON, _ := json.MarshalIndent(gotBody, "", "  ")
		wantJSON, _ := json.MarshalIndent(want, "", "  ")
		t.Errorf("request body mismatch\ngot:\n%s\nwant:\n%s", gotJSON, wantJSON)
	}
}

func TestSendMessages_ParsesOutput(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"id": "resp_2",
			"object": "response",
			"status": "completed",
			"model": "deepseek-flash",
			"output": [
				{"type": "reasoning", "id": "rs_1", "status": "completed",
				 "content": [{"type": "reasoning_text", "text": "thinking..."}]},
				{"type": "message", "id": "msg_1", "status": "completed", "role": "assistant",
				 "content": [{"type": "output_text", "text": "I'll run it."}]},
				{"type": "function_call", "id": "fc_1", "status": "completed",
				 "call_id": "call_9", "name": "bash", "arguments": "{\"command\":\"echo hi\"}"}
			],
			"usage": {"input_tokens": 120, "output_tokens": 45, "total_tokens": 165}
		}`))
	})

	resp, err := client.SendMessages(
		context.Background(), agentloop.ModelDeepseekFlash,
		agentloop.Message{Role: agentloop.MessageRoleSystem, Content: "sys"},
		[]agentloop.Message{{Role: agentloop.MessageRoleUser, Content: "hi"}},
		nil, nil,
	)
	if err != nil {
		t.Fatalf("SendMessages: %v", err)
	}

	want := []agentloop.MessagesBlock{
		{ID: "rs_1", Type: agentloop.MessagesBlockTypeReasoning, Text: "thinking..."},
		{ID: "msg_1", Type: agentloop.MessagesBlockTypeText, Text: "I'll run it."},
		{ID: "call_9", Type: agentloop.MessagesBlockTypeToolUse, Name: "bash", Input: map[string]interface{}{"command": "echo hi"}},
	}
	if !reflect.DeepEqual(resp.GetContent(), want) {
		t.Errorf("content = %+v, want %+v", resp.GetContent(), want)
	}

	toolUses := resp.GetToolUsesBlocks()
	if len(toolUses) != 1 || toolUses[0].ID != "call_9" || toolUses[0].Input["command"] != "echo hi" {
		t.Errorf("tool uses = %+v", toolUses)
	}
	if resp.Usage != (agentloop.Usage{InputTokens: 120, OutputTokens: 45}) {
		t.Errorf("usage = %+v, want 120/45", resp.Usage)
	}
}

func TestSendMessages_HTTPError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Authentication Fails"}}`))
	})

	_, err := client.SendMessages(
		context.Background(), agentloop.ModelDeepseekFlash,
		agentloop.Message{Content: "sys"},
		[]agentloop.Message{{Role: agentloop.MessageRoleUser, Content: "hi"}},
		nil, nil,
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 401") || !strings.Contains(err.Error(), "Authentication Fails") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendMessages_FailedStatus(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"resp_3","status":"failed","error":{"code":"server_error","message":"boom"},"output":[]}`))
	})

	_, err := client.SendMessages(
		context.Background(), agentloop.ModelDeepseekFlash,
		agentloop.Message{Content: "sys"},
		[]agentloop.Message{{Role: agentloop.MessageRoleUser, Content: "hi"}},
		nil, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "server_error: boom") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendMessages_UnsupportedContent(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called")
	})

	_, err := client.SendMessages(
		context.Background(), agentloop.ModelDeepseekFlash,
		agentloop.Message{Content: "sys"},
		[]agentloop.Message{{Role: agentloop.MessageRoleUser, Content: 42}},
		nil, nil,
	)
	if err == nil {
		t.Fatal("expected error for unsupported content type")
	}
}

func TestNewDeepseekClient_DefaultBaseURL(t *testing.T) {
	c := NewDeepseekClient(NewDeepseekClientOptions(agentloop.ModelDeepseekV4Pro)).(*deepseekClient)
	if c.responseBaseURL != "https://api.deepseek.com" {
		t.Errorf("responseBaseURL = %q", c.responseBaseURL)
	}
	if c.GetModel() != agentloop.ModelDeepseekV4Pro {
		t.Errorf("model = %q", c.GetModel())
	}
}
