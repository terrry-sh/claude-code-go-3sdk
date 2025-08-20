package claudesdk

import (
	"context"
	"fmt"
	"os"
)

// Client for bidirectional, interactive conversations with Claude Code.
//
// This client provides full control over the conversation flow with support
// for streaming, interrupts, and dynamic message sending. For simple one-shot
// queries, consider using the Query function instead.
//
// Key features:
//   - Bidirectional: Send and receive messages at any time
//   - Stateful: Maintains conversation context across messages  
//   - Interactive: Send follow-ups based on responses
//   - Control flow: Support for interrupts and session management
//
// When to use Client:
//   - Building chat interfaces or conversational UIs
//   - Interactive debugging or exploration sessions
//   - Multi-turn conversations with context
//   - When you need to react to Claude's responses
//   - Real-time applications with user input
//   - When you need interrupt capabilities
//
// When to use Query() instead:
//   - Simple one-off questions
//   - Batch processing of prompts
//   - Fire-and-forget automation scripts
//   - When all inputs are known upfront
//   - Stateless operations
type Client struct {
	options   *ClaudeCodeOptions
	transport Transport
	connected bool
}

// NewClient creates a new Claude SDK client
func NewClient(options *ClaudeCodeOptions) *Client {
	if options == nil {
		options = NewClaudeCodeOptions()
	}
	
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go-client")
	
	return &Client{
		options: options,
	}
}

// Connect connects to Claude with a prompt or message stream
// If prompt is nil, connects with an empty stream for interactive use
func (c *Client) Connect(ctx context.Context, prompt interface{}) error {
	if c.connected {
		return nil
	}

	// Auto-connect with empty channel if no prompt is provided
	if prompt == nil {
		emptyChan := make(chan map[string]interface{})
		close(emptyChan) // Close immediately to indicate empty stream
		prompt = emptyChan
	}

	// Create subprocess transport
	t, err := NewSubprocessCLITransport(prompt, c.options, "", false)
	if err != nil {
		return err
	}

	c.transport = t
	
	if err := c.transport.Connect(); err != nil {
		return err
	}

	c.connected = true
	return nil
}

// ReceiveMessages receives all messages from Claude
func (c *Client) ReceiveMessages(ctx context.Context) (<-chan Message, error) {
	if !c.connected {
		return nil, NewCLIConnectionError("Not connected. Call Connect() first.")
	}

	dataChan, err := c.transport.ReceiveMessages()
	if err != nil {
		return nil, err
	}

	msgChan := make(chan Message)
	
	go func() {
		defer close(msgChan)
		
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-dataChan:
				if !ok {
					return
				}
				
				// Convert MessageData to map for parser
				dataMap := messageDataToMap(data)
				msg, err := ParseMessage(dataMap)
				if err != nil {
					// Log error and continue
					continue
				}
				
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return msgChan, nil
}

// Query sends a new request in streaming mode
func (c *Client) Query(ctx context.Context, prompt interface{}, sessionID string) error {
	if !c.connected {
		return NewCLIConnectionError("Not connected. Call Connect() first.")
	}

	if sessionID == "" {
		sessionID = "default"
	}

	var messages []MessageData

	switch p := prompt.(type) {
	case string:
		// Handle string prompts
		messages = []MessageData{
			{
				Type: "user",
				Message: map[string]interface{}{
					"role":    "user",
					"content": p,
				},
				ParentToolUseID: nil,
				SessionID:       sessionID,
			},
		}
	case chan map[string]interface{}:
		// Handle channel prompts
		for msg := range p {
			if _, ok := msg["session_id"]; !ok {
				msg["session_id"] = sessionID
			}
			// Convert map to MessageData
			msgData := mapToMessageData(msg)
			messages = append(messages, msgData)
		}
	case []map[string]interface{}:
		// Handle slice of messages
		for _, msg := range p {
			if _, ok := msg["session_id"]; !ok {
				msg["session_id"] = sessionID
			}
			msgData := mapToMessageData(msg)
			messages = append(messages, msgData)
		}
	default:
		return fmt.Errorf("unsupported prompt type: %T", prompt)
	}

	if len(messages) > 0 {
		return c.transport.SendRequest(messages, map[string]interface{}{
			"session_id": sessionID,
		})
	}

	return nil
}

// Interrupt sends an interrupt signal (only works with streaming mode)
func (c *Client) Interrupt() error {
	if !c.connected {
		return NewCLIConnectionError("Not connected. Call Connect() first.")
	}
	return c.transport.Interrupt()
}

// ReceiveResponse receives messages from Claude until and including a ResultMessage
//
// This iterator yields all messages in sequence and automatically terminates
// after yielding a ResultMessage (which indicates the response is complete).
// It's a convenience method over ReceiveMessages() for single-response workflows.
func (c *Client) ReceiveResponse(ctx context.Context) (<-chan Message, error) {
	allMsgs, err := c.ReceiveMessages(ctx)
	if err != nil {
		return nil, err
	}

	respChan := make(chan Message)
	
	go func() {
		defer close(respChan)
		
		for msg := range allMsgs {
			select {
			case respChan <- msg:
				if _, isResult := msg.(*ResultMessage); isResult {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return respChan, nil
}

// Disconnect disconnects from Claude
func (c *Client) Disconnect() error {
	if c.transport != nil {
		err := c.transport.Disconnect()
		c.transport = nil
		c.connected = false
		return err
	}
	return nil
}

// Close is an alias for Disconnect
func (c *Client) Close() error {
	return c.Disconnect()
}

// Helper functions to convert between map and MessageData

func messageDataToMap(data MessageData) map[string]interface{} {
	result := make(map[string]interface{})
	
	result["type"] = data.Type
	
	if data.Message != nil {
		result["message"] = data.Message
	}
	if data.ParentToolUseID != nil {
		result["parent_tool_use_id"] = *data.ParentToolUseID
	}
	if data.SessionID != "" {
		result["session_id"] = data.SessionID
	}
	if data.Content != nil {
		result["content"] = data.Content
	}
	if data.Model != "" {
		result["model"] = data.Model
	}
	if data.Subtype != "" {
		result["subtype"] = data.Subtype
	}
	if data.Data != nil {
		result["data"] = data.Data
	}
	if data.DurationMS > 0 {
		result["duration_ms"] = data.DurationMS
	}
	if data.DurationAPIMS > 0 {
		result["duration_api_ms"] = data.DurationAPIMS
	}
	result["is_error"] = data.IsError
	if data.NumTurns > 0 {
		result["num_turns"] = data.NumTurns
	}
	if data.TotalCostUSD != nil {
		result["total_cost_usd"] = *data.TotalCostUSD
	}
	if data.Usage != nil {
		result["usage"] = data.Usage
	}
	if data.Result != nil {
		result["result"] = *data.Result
	}
	
	return result
}

func mapToMessageData(m map[string]interface{}) MessageData {
	data := MessageData{}
	
	if v, ok := m["type"].(string); ok {
		data.Type = v
	}
	if v, ok := m["message"]; ok {
		data.Message = v.(map[string]interface{})
	}
	if v, ok := m["parent_tool_use_id"].(string); ok {
		data.ParentToolUseID = &v
	}
	if v, ok := m["session_id"].(string); ok {
		data.SessionID = v
	}
	if v, ok := m["content"]; ok {
		data.Content = v
	}
	if v, ok := m["model"].(string); ok {
		data.Model = v
	}
	if v, ok := m["subtype"].(string); ok {
		data.Subtype = v
	}
	if v, ok := m["data"].(map[string]interface{}); ok {
		data.Data = v
	}
	if v, ok := m["duration_ms"].(int); ok {
		data.DurationMS = v
	}
	if v, ok := m["duration_api_ms"].(int); ok {
		data.DurationAPIMS = v
	}
	if v, ok := m["is_error"].(bool); ok {
		data.IsError = v
	}
	if v, ok := m["num_turns"].(int); ok {
		data.NumTurns = v
	}
	if v, ok := m["total_cost_usd"].(float64); ok {
		data.TotalCostUSD = &v
	}
	if v, ok := m["usage"].(map[string]interface{}); ok {
		data.Usage = v
	}
	if v, ok := m["result"].(string); ok {
		data.Result = &v
	}
	
	return data
}