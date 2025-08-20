package claudesdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageTypes(t *testing.T) {
	t.Run("UserMessage creation", func(t *testing.T) {
		msg := &UserMessage{Content: "Hello, Claude!"}
		assert.Equal(t, "Hello, Claude!", msg.Content)
	})

	t.Run("AssistantMessage with text", func(t *testing.T) {
		textBlock := &TextBlock{Text: "Hello, human!"}
		msg := &AssistantMessage{
			Content: []ContentBlock{textBlock},
			Model:   "claude-opus-4-1-20250805",
		}
		assert.Len(t, msg.Content, 1)
		assert.Equal(t, "Hello, human!", msg.Content[0].(*TextBlock).Text)
	})

	t.Run("AssistantMessage with thinking", func(t *testing.T) {
		thinkingBlock := &ThinkingBlock{
			Thinking:  "I'm thinking...",
			Signature: "sig-123",
		}
		msg := &AssistantMessage{
			Content: []ContentBlock{thinkingBlock},
			Model:   "claude-opus-4-1-20250805",
		}
		assert.Len(t, msg.Content, 1)
		assert.Equal(t, "I'm thinking...", msg.Content[0].(*ThinkingBlock).Thinking)
		assert.Equal(t, "sig-123", msg.Content[0].(*ThinkingBlock).Signature)
	})

	t.Run("ToolUseBlock", func(t *testing.T) {
		block := &ToolUseBlock{
			ID:   "tool-123",
			Name: "Read",
			Input: map[string]interface{}{
				"file_path": "/test.txt",
			},
		}
		assert.Equal(t, "tool-123", block.ID)
		assert.Equal(t, "Read", block.Name)
		assert.Equal(t, "/test.txt", block.Input["file_path"])
	})

	t.Run("ToolResultBlock", func(t *testing.T) {
		isError := false
		block := &ToolResultBlock{
			ToolUseID: "tool-123",
			Content:   "File contents here",
			IsError:   &isError,
		}
		assert.Equal(t, "tool-123", block.ToolUseID)
		assert.Equal(t, "File contents here", block.Content)
		assert.False(t, *block.IsError)
	})

	t.Run("ResultMessage", func(t *testing.T) {
		cost := 0.01
		msg := &ResultMessage{
			Subtype:       "success",
			DurationMS:    1500,
			DurationAPIMS: 1200,
			IsError:       false,
			NumTurns:      1,
			SessionID:     "session-123",
			TotalCostUSD:  &cost,
		}
		assert.Equal(t, "success", msg.Subtype)
		assert.Equal(t, 0.01, *msg.TotalCostUSD)
		assert.Equal(t, "session-123", msg.SessionID)
	})
}

func TestOptions(t *testing.T) {
	t.Run("Default options", func(t *testing.T) {
		options := NewClaudeCodeOptions()
		assert.Empty(t, options.AllowedTools)
		assert.Equal(t, 8000, options.MaxThinkingTokens)
		assert.Nil(t, options.SystemPrompt)
		assert.Nil(t, options.PermissionMode)
		assert.False(t, options.ContinueConversation)
		assert.Empty(t, options.DisallowedTools)
	})

	t.Run("Options with tools", func(t *testing.T) {
		options := &ClaudeCodeOptions{
			AllowedTools:    []string{"Read", "Write", "Edit"},
			DisallowedTools: []string{"Bash"},
		}
		assert.Equal(t, []string{"Read", "Write", "Edit"}, options.AllowedTools)
		assert.Equal(t, []string{"Bash"}, options.DisallowedTools)
	})

	t.Run("Options with permission mode", func(t *testing.T) {
		bypass := PermissionModeBypassPermissions
		options := &ClaudeCodeOptions{
			PermissionMode: &bypass,
		}
		assert.Equal(t, PermissionModeBypassPermissions, *options.PermissionMode)

		plan := PermissionModePlan
		optionsPlan := &ClaudeCodeOptions{
			PermissionMode: &plan,
		}
		assert.Equal(t, PermissionModePlan, *optionsPlan.PermissionMode)

		defaultMode := PermissionModeDefault
		optionsDefault := &ClaudeCodeOptions{
			PermissionMode: &defaultMode,
		}
		assert.Equal(t, PermissionModeDefault, *optionsDefault.PermissionMode)

		accept := PermissionModeAcceptEdits
		optionsAccept := &ClaudeCodeOptions{
			PermissionMode: &accept,
		}
		assert.Equal(t, PermissionModeAcceptEdits, *optionsAccept.PermissionMode)
	})

	t.Run("Options with system prompt", func(t *testing.T) {
		systemPrompt := "You are a helpful assistant."
		appendPrompt := "Be concise."
		options := &ClaudeCodeOptions{
			SystemPrompt:       &systemPrompt,
			AppendSystemPrompt: &appendPrompt,
		}
		assert.Equal(t, "You are a helpful assistant.", *options.SystemPrompt)
		assert.Equal(t, "Be concise.", *options.AppendSystemPrompt)
	})

	t.Run("Options with session continuation", func(t *testing.T) {
		resume := "session-123"
		options := &ClaudeCodeOptions{
			ContinueConversation: true,
			Resume:               &resume,
		}
		assert.True(t, options.ContinueConversation)
		assert.Equal(t, "session-123", *options.Resume)
	})

	t.Run("Options with model specification", func(t *testing.T) {
		model := "claude-3-5-sonnet-20241022"
		toolName := "CustomTool"
		options := &ClaudeCodeOptions{
			Model:                    &model,
			PermissionPromptToolName: &toolName,
		}
		assert.Equal(t, "claude-3-5-sonnet-20241022", *options.Model)
		assert.Equal(t, "CustomTool", *options.PermissionPromptToolName)
	})
}