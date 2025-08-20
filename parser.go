package claudesdk

import (
	"encoding/json"
	"fmt"
)

// ParseMessage parses a message from CLI output into typed Message objects
func ParseMessage(data map[string]interface{}) (Message, error) {
	messageType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("message missing 'type' field")
	}

	switch messageType {
	case "user":
		return parseUserMessage(data)
	case "assistant":
		return parseAssistantMessage(data)
	case "system":
		return parseSystemMessage(data)
	case "result":
		return parseResultMessage(data)
	default:
		return nil, fmt.Errorf("unknown message type: %s", messageType)
	}
}

func parseUserMessage(data map[string]interface{}) (*UserMessage, error) {
	messageData, ok := data["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing 'message' field in user message")
	}

	content := messageData["content"]
	
	// Check if content is a list of blocks
	if contentList, ok := content.([]interface{}); ok {
		blocks := []ContentBlock{}
		for _, item := range contentList {
			block, err := parseContentBlock(item)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
		return &UserMessage{Content: blocks}, nil
	}
	
	// Otherwise it's a string
	return &UserMessage{Content: content}, nil
}

func parseAssistantMessage(data map[string]interface{}) (*AssistantMessage, error) {
	messageData, ok := data["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing 'message' field in assistant message")
	}

	model, ok := messageData["model"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'model' field in assistant message")
	}

	contentData, ok := messageData["content"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'content' field in assistant message")
	}

	blocks := []ContentBlock{}
	for _, item := range contentData {
		block, err := parseContentBlock(item)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return &AssistantMessage{
		Content: blocks,
		Model:   model,
	}, nil
}

func parseContentBlock(item interface{}) (ContentBlock, error) {
	blockData, ok := item.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid content block format")
	}

	blockType, ok := blockData["type"].(string)
	if !ok {
		return nil, fmt.Errorf("content block missing 'type' field")
	}

	switch blockType {
	case "text":
		text, ok := blockData["text"].(string)
		if !ok {
			return nil, fmt.Errorf("text block missing 'text' field")
		}
		return &TextBlock{Text: text}, nil

	case "thinking":
		thinking, ok := blockData["thinking"].(string)
		if !ok {
			return nil, fmt.Errorf("thinking block missing 'thinking' field")
		}
		signature, ok := blockData["signature"].(string)
		if !ok {
			return nil, fmt.Errorf("thinking block missing 'signature' field")
		}
		return &ThinkingBlock{
			Thinking:  thinking,
			Signature: signature,
		}, nil

	case "tool_use":
		id, ok := blockData["id"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_use block missing 'id' field")
		}
		name, ok := blockData["name"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_use block missing 'name' field")
		}
		input, ok := blockData["input"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("tool_use block missing 'input' field")
		}
		return &ToolUseBlock{
			ID:    id,
			Name:  name,
			Input: input,
		}, nil

	case "tool_result":
		toolUseID, ok := blockData["tool_use_id"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_result block missing 'tool_use_id' field")
		}
		
		result := &ToolResultBlock{
			ToolUseID: toolUseID,
		}
		
		if content, exists := blockData["content"]; exists {
			result.Content = content
		}
		
		if isError, exists := blockData["is_error"]; exists {
			if b, ok := isError.(bool); ok {
				result.IsError = &b
			}
		}
		
		return result, nil

	default:
		return nil, fmt.Errorf("unknown content block type: %s", blockType)
	}
}

func parseSystemMessage(data map[string]interface{}) (*SystemMessage, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'subtype' field in system message")
	}

	return &SystemMessage{
		Subtype: subtype,
		Data:    data,
	}, nil
}

func parseResultMessage(data map[string]interface{}) (*ResultMessage, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'subtype' field in result message")
	}

	durationMS, ok := getInt(data, "duration_ms")
	if !ok {
		return nil, fmt.Errorf("missing 'duration_ms' field in result message")
	}

	durationAPIMS, ok := getInt(data, "duration_api_ms")
	if !ok {
		return nil, fmt.Errorf("missing 'duration_api_ms' field in result message")
	}

	isError, ok := data["is_error"].(bool)
	if !ok {
		return nil, fmt.Errorf("missing 'is_error' field in result message")
	}

	numTurns, ok := getInt(data, "num_turns")
	if !ok {
		return nil, fmt.Errorf("missing 'num_turns' field in result message")
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'session_id' field in result message")
	}

	msg := &ResultMessage{
		Subtype:       subtype,
		DurationMS:    durationMS,
		DurationAPIMS: durationAPIMS,
		IsError:       isError,
		NumTurns:      numTurns,
		SessionID:     sessionID,
	}

	// Optional fields
	if totalCost, exists := data["total_cost_usd"]; exists {
		if f, ok := totalCost.(float64); ok {
			msg.TotalCostUSD = &f
		}
	}

	if usage, exists := data["usage"]; exists {
		if u, ok := usage.(map[string]interface{}); ok {
			msg.Usage = u
		}
	}

	if result, exists := data["result"]; exists {
		if r, ok := result.(string); ok {
			msg.Result = &r
		}
	}

	return msg, nil
}

// getInt safely extracts an integer from a map, handling both int and float64 types
func getInt(data map[string]interface{}, key string) (int, bool) {
	val, exists := data[key]
	if !exists {
		return 0, false
	}
	
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// ParseMessageFromJSON parses a JSON byte array into a Message
func ParseMessageFromJSON(jsonData []byte) (Message, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return ParseMessage(data)
}