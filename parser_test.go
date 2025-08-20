package claudesdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUserMessage(t *testing.T) {
	t.Run("String content", func(t *testing.T) {
		data := map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"role":    "user",
				"content": "Hello, Claude!",
			},
		}

		msg, err := ParseMessage(data)
		require.NoError(t, err)

		userMsg, ok := msg.(*UserMessage)
		require.True(t, ok)
		assert.Equal(t, "Hello, Claude!", userMsg.Content)
	})

	t.Run("Block content", func(t *testing.T) {
		data := map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Hello",
					},
					map[string]interface{}{
						"type": "tool_use",
						"id":   "tool-123",
						"name": "Read",
						"input": map[string]interface{}{
							"file_path": "/test.txt",
						},
					},
				},
			},
		}

		msg, err := ParseMessage(data)
		require.NoError(t, err)

		userMsg, ok := msg.(*UserMessage)
		require.True(t, ok)

		blocks, ok := userMsg.Content.([]ContentBlock)
		require.True(t, ok)
		assert.Len(t, blocks, 2)

		textBlock, ok := blocks[0].(*TextBlock)
		require.True(t, ok)
		assert.Equal(t, "Hello", textBlock.Text)

		toolBlock, ok := blocks[1].(*ToolUseBlock)
		require.True(t, ok)
		assert.Equal(t, "tool-123", toolBlock.ID)
		assert.Equal(t, "Read", toolBlock.Name)
	})
}

func TestParseAssistantMessage(t *testing.T) {
	data := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"model": "claude-opus-4-1-20250805",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello, human!",
				},
				map[string]interface{}{
					"type":      "thinking",
					"thinking":  "Processing...",
					"signature": "sig-456",
				},
			},
		},
	}

	msg, err := ParseMessage(data)
	require.NoError(t, err)

	assistantMsg, ok := msg.(*AssistantMessage)
	require.True(t, ok)
	assert.Equal(t, "claude-opus-4-1-20250805", assistantMsg.Model)
	assert.Len(t, assistantMsg.Content, 2)

	textBlock, ok := assistantMsg.Content[0].(*TextBlock)
	require.True(t, ok)
	assert.Equal(t, "Hello, human!", textBlock.Text)

	thinkingBlock, ok := assistantMsg.Content[1].(*ThinkingBlock)
	require.True(t, ok)
	assert.Equal(t, "Processing...", thinkingBlock.Thinking)
	assert.Equal(t, "sig-456", thinkingBlock.Signature)
}

func TestParseSystemMessage(t *testing.T) {
	data := map[string]interface{}{
		"type":    "system",
		"subtype": "info",
		"data": map[string]interface{}{
			"message": "System information",
		},
	}

	msg, err := ParseMessage(data)
	require.NoError(t, err)

	systemMsg, ok := msg.(*SystemMessage)
	require.True(t, ok)
	assert.Equal(t, "info", systemMsg.Subtype)
	assert.Equal(t, "System information", systemMsg.Data["data"].(map[string]interface{})["message"])
}

func TestParseResultMessage(t *testing.T) {
	data := map[string]interface{}{
		"type":            "result",
		"subtype":         "success",
		"duration_ms":     1500,
		"duration_api_ms": 1200,
		"is_error":        false,
		"num_turns":       1,
		"session_id":      "session-123",
		"total_cost_usd":  0.01,
		"usage": map[string]interface{}{
			"input_tokens":  float64(100),
			"output_tokens": float64(50),
		},
		"result": "completed",
	}

	msg, err := ParseMessage(data)
	require.NoError(t, err)

	resultMsg, ok := msg.(*ResultMessage)
	require.True(t, ok)
	assert.Equal(t, "success", resultMsg.Subtype)
	assert.Equal(t, 1500, resultMsg.DurationMS)
	assert.Equal(t, 1200, resultMsg.DurationAPIMS)
	assert.False(t, resultMsg.IsError)
	assert.Equal(t, 1, resultMsg.NumTurns)
	assert.Equal(t, "session-123", resultMsg.SessionID)
	assert.NotNil(t, resultMsg.TotalCostUSD)
	assert.Equal(t, 0.01, *resultMsg.TotalCostUSD)
	assert.NotNil(t, resultMsg.Usage)
	assert.Equal(t, float64(100), resultMsg.Usage["input_tokens"])
	assert.NotNil(t, resultMsg.Result)
	assert.Equal(t, "completed", *resultMsg.Result)
}

func TestParseContentBlocks(t *testing.T) {
	t.Run("ToolResultBlock", func(t *testing.T) {
		blockData := map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": "tool-789",
			"content":     "File not found",
			"is_error":    true,
		}

		block, err := parseContentBlock(blockData)
		require.NoError(t, err)

		toolResult, ok := block.(*ToolResultBlock)
		require.True(t, ok)
		assert.Equal(t, "tool-789", toolResult.ToolUseID)
		assert.Equal(t, "File not found", toolResult.Content)
		assert.NotNil(t, toolResult.IsError)
		assert.True(t, *toolResult.IsError)
	})
}

func TestParseErrors(t *testing.T) {
	t.Run("Missing type field", func(t *testing.T) {
		data := map[string]interface{}{
			"message": "test",
		}

		_, err := ParseMessage(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'type' field")
	})

	t.Run("Unknown message type", func(t *testing.T) {
		data := map[string]interface{}{
			"type": "unknown",
		}

		_, err := ParseMessage(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown message type")
	})

	t.Run("Missing required fields", func(t *testing.T) {
		data := map[string]interface{}{
			"type": "assistant",
			"message": map[string]interface{}{
				// Missing model field
				"content": []interface{}{},
			},
		}

		_, err := ParseMessage(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'model' field")
	})
}