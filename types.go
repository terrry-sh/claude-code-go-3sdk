package claudesdk

import (
	"encoding/json"
)

// PermissionMode represents different permission modes for Claude
type PermissionMode string

const (
	PermissionModeDefault          PermissionMode = "default"
	PermissionModeAcceptEdits      PermissionMode = "acceptEdits"
	PermissionModePlan             PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// MCPServerType represents the type of MCP server
type MCPServerType string

const (
	MCPServerTypeStdio MCPServerType = "stdio"
	MCPServerTypeSSE   MCPServerType = "sse"
	MCPServerTypeHTTP  MCPServerType = "http"
)

// MCPStdioServerConfig represents MCP stdio server configuration
type MCPStdioServerConfig struct {
	Type    MCPServerType     `json:"type,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCPSSEServerConfig represents MCP SSE server configuration
type MCPSSEServerConfig struct {
	Type    MCPServerType     `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// MCPHTTPServerConfig represents MCP HTTP server configuration
type MCPHTTPServerConfig struct {
	Type    MCPServerType     `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// MCPServerConfig represents any MCP server configuration
type MCPServerConfig interface {
	isMCPServerConfig()
}

func (MCPStdioServerConfig) isMCPServerConfig() {}
func (MCPSSEServerConfig) isMCPServerConfig()   {}
func (MCPHTTPServerConfig) isMCPServerConfig()  {}

// ContentBlock represents a block of content in a message
type ContentBlock interface {
	isContentBlock()
}

// TextBlock represents text content
type TextBlock struct {
	Text string `json:"text"`
}

func (TextBlock) isContentBlock() {}

// ThinkingBlock represents thinking content
type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

func (ThinkingBlock) isContentBlock() {}

// ToolUseBlock represents tool use content
type ToolUseBlock struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (ToolUseBlock) isContentBlock() {}

// ToolResultBlock represents tool result content
type ToolResultBlock struct {
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content,omitempty"`
	IsError   *bool       `json:"is_error,omitempty"`
}

func (ToolResultBlock) isContentBlock() {}

// Message represents any message type
type Message interface {
	isMessage()
}

// UserMessage represents a user message
type UserMessage struct {
	Content interface{} `json:"content"` // string or []ContentBlock
}

func (UserMessage) isMessage() {}

// AssistantMessage represents an assistant message with content blocks
type AssistantMessage struct {
	Content []ContentBlock `json:"content"`
	Model   string         `json:"model"`
}

func (AssistantMessage) isMessage() {}

// SystemMessage represents a system message with metadata
type SystemMessage struct {
	Subtype string                 `json:"subtype"`
	Data    map[string]interface{} `json:"data"`
}

func (SystemMessage) isMessage() {}

// ResultMessage represents a result message with cost and usage information
type ResultMessage struct {
	Subtype        string                 `json:"subtype"`
	DurationMS     int                    `json:"duration_ms"`
	DurationAPIMS  int                    `json:"duration_api_ms"`
	IsError        bool                   `json:"is_error"`
	NumTurns       int                    `json:"num_turns"`
	SessionID      string                 `json:"session_id"`
	TotalCostUSD   *float64               `json:"total_cost_usd,omitempty"`
	Usage          map[string]interface{} `json:"usage,omitempty"`
	Result         *string                `json:"result,omitempty"`
}

func (ResultMessage) isMessage() {}

// ClaudeCodeOptions represents query options for Claude SDK
type ClaudeCodeOptions struct {
	AllowedTools              []string                   `json:"allowed_tools,omitempty"`
	MaxThinkingTokens         int                        `json:"max_thinking_tokens,omitempty"`
	SystemPrompt              *string                    `json:"system_prompt,omitempty"`
	AppendSystemPrompt        *string                    `json:"append_system_prompt,omitempty"`
	MCPServers                map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
	MCPServersPath            *string                    `json:"-"` // For file path to MCP config
	PermissionMode            *PermissionMode            `json:"permission_mode,omitempty"`
	ContinueConversation      bool                       `json:"continue_conversation,omitempty"`
	Resume                    *string                    `json:"resume,omitempty"`
	MaxTurns                  *int                       `json:"max_turns,omitempty"`
	DisallowedTools           []string                   `json:"disallowed_tools,omitempty"`
	Model                     *string                    `json:"model,omitempty"`
	PermissionPromptToolName  *string                    `json:"permission_prompt_tool_name,omitempty"`
	CWD                       *string                    `json:"cwd,omitempty"`
	Settings                  *string                    `json:"settings,omitempty"`
	AddDirs                   []string                   `json:"add_dirs,omitempty"`
	ExtraArgs                 map[string]*string         `json:"-"` // Pass arbitrary CLI flags
}

// NewClaudeCodeOptions creates a new ClaudeCodeOptions with defaults
func NewClaudeCodeOptions() *ClaudeCodeOptions {
	return &ClaudeCodeOptions{
		AllowedTools:      []string{},
		MaxThinkingTokens: 8000,
		MCPServers:        make(map[string]MCPServerConfig),
		DisallowedTools:   []string{},
		AddDirs:           []string{},
		ExtraArgs:         make(map[string]*string),
	}
}

// MessageData represents the structure of messages sent to/from Claude
type MessageData struct {
	Type             string                 `json:"type"`
	Message          map[string]interface{} `json:"message,omitempty"`
	ParentToolUseID  *string                `json:"parent_tool_use_id,omitempty"`
	SessionID        string                 `json:"session_id,omitempty"`
	Content          interface{}            `json:"content,omitempty"`
	Model            string                 `json:"model,omitempty"`
	Subtype          string                 `json:"subtype,omitempty"`
	Data             map[string]interface{} `json:"data,omitempty"`
	DurationMS       int                    `json:"duration_ms,omitempty"`
	DurationAPIMS    int                    `json:"duration_api_ms,omitempty"`
	IsError          bool                   `json:"is_error,omitempty"`
	NumTurns         int                    `json:"num_turns,omitempty"`
	TotalCostUSD     *float64               `json:"total_cost_usd,omitempty"`
	Usage            map[string]interface{} `json:"usage,omitempty"`
	Result           *string                `json:"result,omitempty"`
}

// Transport defines the interface for communication with Claude
type Transport interface {
	Connect() error
	Disconnect() error
	SendRequest(messages []MessageData, metadata map[string]interface{}) error
	ReceiveMessages() (<-chan MessageData, error)
	Interrupt() error
}

// Custom JSON marshaling for messages
func (m UserMessage) MarshalJSON() ([]byte, error) {
	type Alias UserMessage
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  "user",
		Alias: (*Alias)(&m),
	})
}

func (m AssistantMessage) MarshalJSON() ([]byte, error) {
	type Alias AssistantMessage
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  "assistant",
		Alias: (*Alias)(&m),
	})
}

func (m SystemMessage) MarshalJSON() ([]byte, error) {
	type Alias SystemMessage
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  "system",
		Alias: (*Alias)(&m),
	})
}

func (m ResultMessage) MarshalJSON() ([]byte, error) {
	type Alias ResultMessage
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  "result",
		Alias: (*Alias)(&m),
	})
}