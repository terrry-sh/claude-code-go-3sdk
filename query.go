package claudesdk

import (
	"context"
	"os"
)

// Query performs a one-shot interaction with Claude Code.
//
// This function is ideal for simple, stateless queries where you don't need
// bidirectional communication or conversation management. For interactive,
// stateful conversations, use Client instead.
//
// Key differences from Client:
//   - Unidirectional: Send all messages upfront, receive all responses
//   - Stateless: Each query is independent, no conversation state
//   - Simple: Fire-and-forget style, no connection management
//   - No interrupts: Cannot interrupt or send follow-up messages
//
// When to use Query:
//   - Simple one-off questions ("What is 2+2?")
//   - Batch processing of independent prompts
//   - Code generation or analysis tasks
//   - Automated scripts and CI/CD pipelines
//   - When you know all inputs upfront
//
// When to use Client:
//   - Interactive conversations with follow-ups
//   - Chat applications or REPL-like interfaces
//   - When you need to send messages based on responses
//   - When you need interrupt capabilities
//   - Long-running sessions with state
//
// Parameters:
//   - ctx: Context for cancellation
//   - prompt: The prompt to send to Claude. Can be a string for single-shot queries
//     or a channel/slice of maps for streaming mode
//   - options: Optional configuration (defaults to NewClaudeCodeOptions() if nil)
//
// Returns a channel of Messages from the conversation
//
// Example - Simple query:
//
//	for msg := range Query(ctx, "What is the capital of France?", nil) {
//	    fmt.Println(msg)
//	}
//
// Example - With options:
//
//	options := &ClaudeCodeOptions{
//	    SystemPrompt: String("You are an expert Python developer"),
//	    CWD: String("/home/user/project"),
//	}
//	for msg := range Query(ctx, "Create a Python web server", options) {
//	    fmt.Println(msg)
//	}
func Query(ctx context.Context, prompt interface{}, options *ClaudeCodeOptions) <-chan Message {
	msgChan := make(chan Message)

	go func() {
		defer close(msgChan)

		if options == nil {
			options = NewClaudeCodeOptions()
		}

		os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go")

		// Create transport with closeStdinAfterPrompt=true for one-shot mode
		t, err := NewSubprocessCLITransport(prompt, options, "", true)
		if err != nil {
			// Send error as a system message
			msgChan <- &SystemMessage{
				Subtype: "error",
				Data: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}

		// Connect
		if err := t.Connect(); err != nil {
			msgChan <- &SystemMessage{
				Subtype: "error",
				Data: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}
		defer t.Disconnect()

		// Receive messages
		dataChan, err := t.ReceiveMessages()
		if err != nil {
			msgChan <- &SystemMessage{
				Subtype: "error",
				Data: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}

		// Parse and forward messages
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

	return msgChan
}

// QuerySync performs a synchronous query and returns all messages
//
// This is a convenience wrapper around Query that collects all messages
// and returns them as a slice. Useful when you want all results at once.
//
// Example:
//
//	messages, err := QuerySync(ctx, "What is 2+2?", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, msg := range messages {
//	    fmt.Println(msg)
//	}
func QuerySync(ctx context.Context, prompt interface{}, options *ClaudeCodeOptions) ([]Message, error) {
	var messages []Message
	
	for msg := range Query(ctx, prompt, options) {
		messages = append(messages, msg)
		
		// Check if this is an error message
		if sysMsg, ok := msg.(*SystemMessage); ok && sysMsg.Subtype == "error" {
			if errStr, ok := sysMsg.Data["error"].(string); ok {
				return messages, NewCLIConnectionError(errStr)
			}
		}
	}
	
	return messages, nil
}

// Helper function to create string pointers (useful for options)
func String(s string) *string {
	return &s
}

// Helper function to create int pointers (useful for options)
func Int(i int) *int {
	return &i
}

// Helper function to create float64 pointers (useful for options)
func Float64(f float64) *float64 {
	return &f
}

// Helper function to create PermissionMode pointers
func Permission(p PermissionMode) *PermissionMode {
	return &p
}