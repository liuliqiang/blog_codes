package agentloop

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/liuliqiang/log4go"
)

// mcpOut is where MCP activity is printed; tests swap it.
var mcpOut io.Writer = os.Stdout

// MCPServerConfig declares one server the model may connect to. Command is run as a child process and spoken to over
// stdio, which is the MCP transport for local servers.
type MCPServerConfig struct {
	Name    string
	Command string
	Args    []string
	Env     []string // extra KEY=VALUE entries on top of the agent's environment
}

// MCPConfig lists the servers that exist and what the host allows their tools to do. Hints a server sends about its
// own tools ("readOnlyHint") are the server's claim, not authorization, so they are never consulted: only this policy
// is. A tool with no entry needs the user's confirmation.
type MCPConfig struct {
	Servers []MCPServerConfig
	Policy  map[string]string // model-facing tool name (mcp__server__tool) → "allow" | "confirm" | "deny"
	Timeout time.Duration     // per request
}

// MCP is the active config.
var MCP = MCPConfig{Timeout: 30 * time.Second}

const (
	mcpPolicyAllow   = "allow"
	mcpPolicyConfirm = "confirm"
	mcpPolicyDeny    = "deny"

	mcpProtocolVersion = "2024-11-05"
	// mcpToolNameLimit is the longest tool name the model APIs accept.
	mcpToolNameLimit = 64
)

/* vvvvvvvvvvvvvvvvvvvvv JSON-RPC over stdio vvvvvvvvvvvvvvvvvvvvv */

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return fmt.Sprintf("%s (code %d)", e.Message, e.Code) }

type rpcResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

// mcpTransport carries JSON-RPC calls to one server. It is an interface so tests can talk to a server over a pipe
// instead of spawning a process.
type mcpTransport interface {
	Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error)
	Notify(method string, params interface{}) error
	Close() error
}

// stdioTransport speaks newline-delimited JSON-RPC to a server's stdin and stdout, which is what MCP's stdio
// transport is. One reader goroutine matches replies to the calls waiting for them by id.
type stdioTransport struct {
	mu      sync.Mutex
	in      io.WriteCloser
	out     io.ReadCloser
	cmd     *exec.Cmd
	nextID  int
	pending map[int]chan rpcResponse
	closed  bool
	done    chan struct{}
}

func newStdioTransport(in io.WriteCloser, out io.ReadCloser, cmd *exec.Cmd) *stdioTransport {
	t := &stdioTransport{in: in, out: out, cmd: cmd, pending: map[int]chan rpcResponse{}, done: make(chan struct{})}
	go t.read()
	return t
}

// startStdioServer launches cfg's command and speaks to it over its stdin and stdout.
func startStdioServer(ctx context.Context, cfg MCPServerConfig) (*stdioTransport, error) {
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("server %s has no command", cfg.Name)
	}
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = append(os.Environ(), cfg.Env...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "mcp stdin pipe failed: %v, server: %s", err, cfg.Name)
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "mcp stdout pipe failed: %v, server: %s", err, cfg.Name)
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		log4go.DefaultLogger().Error(ctx, "start mcp server failed: %v, server: %s, command: %s", err, cfg.Name, cfg.Command)
		return nil, err
	}
	return newStdioTransport(stdin, stdout, cmd), nil
}

// read delivers every reply to whoever is waiting for its id. Server notifications, which carry no id, are ignored.
func (t *stdioTransport) read() {
	defer close(t.done)
	scanner := bufio.NewScanner(t.out)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil || resp.ID == 0 {
			continue
		}
		t.mu.Lock()
		ch, ok := t.pending[resp.ID]
		delete(t.pending, resp.ID)
		t.mu.Unlock()
		if ok {
			ch <- resp
		}
	}
	// the server is gone: release everyone still waiting
	t.mu.Lock()
	for id, ch := range t.pending {
		close(ch)
		delete(t.pending, id)
	}
	t.closed = true
	t.mu.Unlock()
}

func (t *stdioTransport) write(payload interface{}) error {
	line, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return errors.New("mcp server connection is closed")
	}
	_, err = t.in.Write(append(line, '\n'))
	return err
}

func (t *stdioTransport) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, errors.New("mcp server connection is closed")
	}
	t.nextID++
	id := t.nextID
	ch := make(chan rpcResponse, 1)
	t.pending[id] = ch
	t.mu.Unlock()

	if err := t.write(rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}); err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		log4go.DefaultLogger().Error(ctx, "mcp request failed: %v, method: %s", err, method)
		return nil, err
	}

	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("mcp server closed the connection during %s", method)
		}
		if resp.Error != nil {
			log4go.DefaultLogger().Error(ctx, "mcp server returned an error: %v, method: %s", resp.Error, method)
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		log4go.DefaultLogger().Error(ctx, "mcp request cancelled: %v, method: %s", ctx.Err(), method)
		return nil, ctx.Err()
	}
}

func (t *stdioTransport) Notify(method string, params interface{}) error {
	return t.write(rpcRequest{JSONRPC: "2.0", Method: method, Params: params})
}

func (t *stdioTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	in := t.in
	t.mu.Unlock()

	_ = in.Close() // a well-behaved server exits when its stdin closes
	if t.cmd == nil {
		_ = t.out.Close()
		return nil
	}
	select {
	case <-t.done:
	case <-time.After(2 * time.Second):
		_ = t.cmd.Process.Kill()
	}
	_ = t.cmd.Wait()
	return nil
}

/* vvvvvvvvvvvvvvvvvvvvv client vvvvvvvvvvvvvvvvvvvvv */

// MCPToolDef is one tool as the server described it.
type MCPToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPClient is one connected server: what it said it can do, and the way to ask it to do so.
type MCPClient struct {
	name      string
	transport mcpTransport

	mu    sync.Mutex
	tools []MCPToolDef
}

// connectMCP performs the MCP handshake and discovery: initialize, initialized, then tools/list.
func connectMCP(ctx context.Context, name string, transport mcpTransport) (*MCPClient, error) {
	c := &MCPClient{name: name, transport: transport}
	initParams := map[string]interface{}{
		"protocolVersion": mcpProtocolVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "llmagent", "version": "0.1"},
	}
	if _, err := transport.Call(ctx, "initialize", initParams); err != nil {
		return nil, fmt.Errorf("initialize %s: %w", name, err)
	}
	if err := transport.Notify("notifications/initialized", map[string]interface{}{}); err != nil {
		log4go.DefaultLogger().Error(ctx, "mcp initialized notification failed: %v, server: %s", err, name)
	}
	if err := c.refreshTools(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// refreshTools re-runs discovery; a server may change what it offers.
func (c *MCPClient) refreshTools(ctx context.Context) error {
	raw, err := c.transport.Call(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("tools/list %s: %w", c.name, err)
	}
	var listed struct {
		Tools []MCPToolDef `json:"tools"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		log4go.DefaultLogger().Error(ctx, "unmarshal tools/list failed: %v, server: %s, result: %s", err, c.name, raw)
		return fmt.Errorf("tools/list %s: %w", c.name, err)
	}
	c.mu.Lock()
	c.tools = listed.Tools
	c.mu.Unlock()
	return nil
}

// Tools is what the server last said it offers.
func (c *MCPClient) Tools() []MCPToolDef {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]MCPToolDef(nil), c.tools...)
}

// CallTool invokes one tool and flattens the reply to text. A tool the server reports as failing comes back as an
// error so the model sees it as a failed tool call rather than a result.
func (c *MCPClient) CallTool(ctx context.Context, tool string, args map[string]interface{}) (string, error) {
	if args == nil {
		args = map[string]interface{}{}
	}
	raw, err := c.transport.Call(ctx, "tools/call", map[string]interface{}{"name": tool, "arguments": args})
	if err != nil {
		return "", err
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		log4go.DefaultLogger().Error(ctx, "unmarshal tools/call failed: %v, server: %s, tool: %s, result: %s", err, c.name, tool, raw)
		return "", err
	}
	var parts []string
	for _, item := range result.Content {
		if item.Text != "" {
			parts = append(parts, item.Text)
		}
	}
	text := strings.Join(parts, "\n")
	if result.IsError {
		if text == "" {
			text = "the server reported an error"
		}
		return "", errors.New(text)
	}
	if text == "" {
		text = "(no content)"
	}
	return text, nil
}

/* vvvvvvvvvvvvvvvvvvvvv registry vvvvvvvvvvvvvvvvvvvvv */

// MCPRegistry holds the servers this process has connected to. It is shared with subagents and teammates: a
// connection belongs to the process, not to one conversation.
type MCPRegistry struct {
	mu      sync.Mutex
	clients map[string]*MCPClient
	order   []string
}

func NewMCPRegistry() *MCPRegistry { return &MCPRegistry{clients: map[string]*MCPClient{}} }

// Connect starts the named server from the config and discovers its tools.
func (r *MCPRegistry) Connect(ctx context.Context, name string) (*MCPClient, error) {
	r.mu.Lock()
	if _, ok := r.clients[name]; ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("mcp server %s is already connected", name)
	}
	r.mu.Unlock()

	var cfg *MCPServerConfig
	for i := range MCP.Servers {
		if MCP.Servers[i].Name == name {
			cfg = &MCP.Servers[i]
			break
		}
	}
	if cfg == nil {
		return nil, fmt.Errorf("unknown mcp server %q; configured: %s", name, strings.Join(configuredMCPServers(), ", "))
	}

	transport, err := startStdioServer(ctx, *cfg)
	if err != nil {
		return nil, err
	}
	handshake, cancel := context.WithTimeout(ctx, MCP.Timeout)
	defer cancel()
	client, err := connectMCP(handshake, name, transport)
	if err != nil {
		_ = transport.Close()
		log4go.DefaultLogger().Error(ctx, "connect mcp server failed: %v, server: %s", err, name)
		return nil, err
	}
	return r.add(name, client)
}

// add registers an already connected client; it is also how tests install a fake server.
func (r *MCPRegistry) add(name string, client *MCPClient) (*MCPClient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.clients[name]; ok {
		return nil, fmt.Errorf("mcp server %s is already connected", name)
	}
	r.clients[name] = client
	r.order = append(r.order, name)
	fmt.Fprintf(mcpOut, "\033[34m[mcp] connected %s with %d tool(s)\033[0m\n", name, len(client.Tools()))
	return client, nil
}

// Names lists the connected servers in connection order.
func (r *MCPRegistry) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.order...)
}

// Close disconnects every server.
func (r *MCPRegistry) Close() {
	r.mu.Lock()
	clients := r.clients
	r.clients, r.order = map[string]*MCPClient{}, nil
	r.mu.Unlock()
	for _, c := range clients {
		_ = c.transport.Close()
	}
}

var mcpNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// normalizeMCPName replaces everything outside the tool-name alphabet with underscores and drops leading and
// trailing separators, so a server-side name like ".get.version." reads as get_version.
func normalizeMCPName(s string) string {
	return strings.Trim(mcpNameUnsafe.ReplaceAllString(s, "_"), "_-")
}

// mcpToolName is the model-facing name of a server's tool. The prefix keeps two servers that both offer "search"
// apart.
func mcpToolName(server, tool string) string {
	return "mcp__" + normalizeMCPName(server) + "__" + normalizeMCPName(tool)
}

// Tools turns every discovered tool into an agent tool. Normalization can map two different server-side names onto
// one model-facing name, so collisions are reported rather than silently resolved; an over-long name is skipped for
// the same reason.
func (r *MCPRegistry) Tools() []Tool {
	r.mu.Lock()
	servers := append([]string(nil), r.order...)
	clients := make([]*MCPClient, 0, len(servers))
	for _, name := range servers {
		clients = append(clients, r.clients[name])
	}
	r.mu.Unlock()

	var tools []Tool
	seen := map[string]string{} // model-facing name → "server/tool" it came from
	for i, client := range clients {
		server := servers[i]
		for _, def := range client.Tools() {
			name := mcpToolName(server, def.Name)
			origin := server + "/" + def.Name
			if previous, clash := seen[name]; clash {
				log4go.DefaultLogger().Error(context.Background(), "mcp tool name collision after normalization: %s and %s both map to %s", previous, origin, name)
				fmt.Fprintf(mcpOut, "\033[34m[mcp] skipped %s: normalized name %s already used by %s\033[0m\n", origin, name, previous)
				continue
			}
			if len(name) > mcpToolNameLimit {
				log4go.DefaultLogger().Error(context.Background(), "mcp tool name %s is longer than %d characters, skipped", name, mcpToolNameLimit)
				fmt.Fprintf(mcpOut, "\033[34m[mcp] skipped %s: name is longer than %d characters\033[0m\n", origin, mcpToolNameLimit)
				continue
			}
			seen[name] = origin
			schema := def.InputSchema
			if schema == nil {
				schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
			}
			description := def.Description
			if description == "" {
				description = fmt.Sprintf("Tool %s from the %s MCP server.", def.Name, server)
			}
			tools = append(tools, Tool{
				Name:        name,
				Description: description,
				InputSchema: schema,
				Handler:     mcpHandler(client, def.Name),
			})
		}
	}
	return tools
}

// mcpHandler binds one client and tool so every handler calls its own tool rather than the last of the loop.
func mcpHandler(client *MCPClient, tool string) ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) (string, error) {
		callCtx, cancel := context.WithTimeout(ctx, MCP.Timeout)
		defer cancel()
		out, err := client.CallTool(callCtx, tool, input)
		if err != nil {
			// the error is returned to the model as a failed tool result; a bad argument or a broken server must not
			// end the loop
			log4go.DefaultLogger().Error(ctx, "mcp tool call failed: %v, server: %s, tool: %s, input: %+v", err, client.name, tool, input)
			return "", fmt.Errorf("mcp %s/%s: %w", client.name, tool, err)
		}
		return out, nil
	}
}

func configuredMCPServers() []string {
	var names []string
	for _, s := range MCP.Servers {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return []string{"(none)"}
	}
	return names
}

/* vvvvvvvvvvvvvvvvvvvvv permission vvvvvvvvvvvvvvvvvvvvv */

// isMCPTool reports whether a tool call is going to an MCP server.
func isMCPTool(name string) bool { return strings.HasPrefix(name, "mcp__") }

// mcpPolicyFor is the host's decision for one MCP tool. Anything the host did not configure needs confirmation:
// the server's own description of the tool is not authorization.
func mcpPolicyFor(name string) string {
	switch MCP.Policy[name] {
	case mcpPolicyAllow:
		return mcpPolicyAllow
	case mcpPolicyDeny:
		return mcpPolicyDeny
	default:
		return mcpPolicyConfirm
	}
}

/* vvvvvvvvvvvvvvvvvvvvv tools vvvvvvvvvvvvvvvvvvvvv */

const mcpSystemPromptGuidance = `External tools live on MCP servers and are not available until you connect: call connect_mcp with a server name, and its tools appear as mcp__<server>__<tool> from your next turn onwards. list_mcp lists what is configured and what is connected.`

func (a *agent) runConnectMCP(ctx context.Context, input map[string]interface{}) (string, error) {
	name, err := stringArg(input, "name")
	if err != nil {
		return "", err
	}
	client, err := a.mcp.Connect(ctx, name)
	if err != nil {
		log4go.DefaultLogger().Error(ctx, "[%s] connect_mcp failed: %v, name: %s", agentNameFrom(ctx), err, name)
		return "", err
	}
	defs := client.Tools()
	if len(defs) == 0 {
		return fmt.Sprintf("Connected to %s; it offers no tools.", name), nil
	}
	var lines []string
	for _, def := range defs {
		lines = append(lines, fmt.Sprintf("- %s: %s", mcpToolName(name, def.Name), oneLine(def.Description)))
	}
	return fmt.Sprintf("Connected to %s. Its tools are available from your next turn:\n%s", name, strings.Join(lines, "\n")), nil
}

func (a *agent) runListMCP(_ context.Context, _ map[string]interface{}) (string, error) {
	connected := map[string]bool{}
	for _, name := range a.mcp.Names() {
		connected[name] = true
	}
	var lines []string
	for _, name := range configuredMCPServers() {
		if name == "(none)" {
			return "No MCP servers are configured.", nil
		}
		state := "not connected"
		if connected[name] {
			state = "connected"
		}
		lines = append(lines, fmt.Sprintf("- %s [%s]", name, state))
	}
	for _, tool := range a.mcp.Tools() {
		lines = append(lines, "  "+tool.Name)
	}
	return strings.Join(lines, "\n"), nil
}
