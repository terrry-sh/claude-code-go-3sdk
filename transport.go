package claudesdk

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const maxBufferSize = 1024 * 1024 // 1MB buffer limit

// SubprocessCLITransport implements Transport using Claude Code CLI subprocess
type SubprocessCLITransport struct {
	prompt                   interface{} // string or chan map[string]interface{}
	isStreaming             bool
	options                 *ClaudeCodeOptions
	cliPath                 string
	cwd                     string
	closeStdinAfterPrompt   bool

	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	stderr      io.ReadCloser
	stderrFile  *os.File
	
	msgChan     chan MessageData
	errChan     chan error
	doneChan    chan struct{}
	
	mu          sync.Mutex
	connected   bool
	requestCounter int
}

// NewSubprocessCLITransport creates a new subprocess transport
func NewSubprocessCLITransport(prompt interface{}, options *ClaudeCodeOptions, cliPath string, closeStdinAfterPrompt bool) (*SubprocessCLITransport, error) {
	if options == nil {
		options = NewClaudeCodeOptions()
	}

	t := &SubprocessCLITransport{
		prompt:                prompt,
		options:              options,
		cliPath:              cliPath,
		closeStdinAfterPrompt: closeStdinAfterPrompt,
		msgChan:              make(chan MessageData, 100),
		errChan:              make(chan error, 1),
		doneChan:             make(chan struct{}),
	}

	// Determine if streaming based on prompt type
	switch p := prompt.(type) {
	case string:
		t.isStreaming = false
	case chan map[string]interface{}:
		t.isStreaming = true
	case <-chan map[string]interface{}:
		t.isStreaming = true
	default:
		return nil, fmt.Errorf("unsupported prompt type: %T", p)
	}

	if t.cliPath == "" {
		path, err := t.findCLI()
		if err != nil {
			return nil, err
		}
		t.cliPath = path
	}

	if options.CWD != nil {
		t.cwd = *options.CWD
	}

	return t, nil
}

func (t *SubprocessCLITransport) findCLI() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}

	// Check common locations
	locations := []string{
		filepath.Join(os.Getenv("HOME"), ".npm-global/bin/claude"),
		"/usr/local/bin/claude",
		filepath.Join(os.Getenv("HOME"), ".local/bin/claude"),
		filepath.Join(os.Getenv("HOME"), "node_modules/.bin/claude"),
		filepath.Join(os.Getenv("HOME"), ".yarn/bin/claude"),
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Check if Node is installed
	if _, err := exec.LookPath("node"); err != nil {
		return "", fmt.Errorf("Claude Code requires Node.js, which is not installed.\n\n" +
			"Install Node.js from: https://nodejs.org/\n" +
			"\nAfter installing Node.js, install Claude Code:\n" +
			"  npm install -g @anthropic-ai/claude-code")
	}

	return "", fmt.Errorf("Claude Code not found. Install with:\n" +
		"  npm install -g @anthropic-ai/claude-code\n" +
		"\nIf already installed locally, try:\n" +
		"  export PATH=\"$HOME/node_modules/.bin:$PATH\"\n" +
		"\nOr specify the path when creating transport")
}

func (t *SubprocessCLITransport) buildCommand() []string {
	cmd := []string{t.cliPath, "--output-format", "stream-json", "--verbose"}

	if t.options.SystemPrompt != nil {
		cmd = append(cmd, "--system-prompt", *t.options.SystemPrompt)
	}

	if t.options.AppendSystemPrompt != nil {
		cmd = append(cmd, "--append-system-prompt", *t.options.AppendSystemPrompt)
	}

	if len(t.options.AllowedTools) > 0 {
		cmd = append(cmd, "--allowedTools", strings.Join(t.options.AllowedTools, ","))
	}

	if t.options.MaxTurns != nil {
		cmd = append(cmd, "--max-turns", fmt.Sprint(*t.options.MaxTurns))
	}

	if len(t.options.DisallowedTools) > 0 {
		cmd = append(cmd, "--disallowedTools", strings.Join(t.options.DisallowedTools, ","))
	}

	if t.options.Model != nil {
		cmd = append(cmd, "--model", *t.options.Model)
	}

	if t.options.PermissionPromptToolName != nil {
		cmd = append(cmd, "--permission-prompt-tool", *t.options.PermissionPromptToolName)
	}

	if t.options.PermissionMode != nil {
		cmd = append(cmd, "--permission-mode", string(*t.options.PermissionMode))
	}

	if t.options.ContinueConversation {
		cmd = append(cmd, "--continue")
	}

	if t.options.Resume != nil {
		cmd = append(cmd, "--resume", *t.options.Resume)
	}

	if t.options.Settings != nil {
		cmd = append(cmd, "--settings", *t.options.Settings)
	}

	for _, dir := range t.options.AddDirs {
		cmd = append(cmd, "--add-dir", dir)
	}

	// Handle MCP servers
	if len(t.options.MCPServers) > 0 {
		mcpConfig := map[string]interface{}{
			"mcpServers": t.options.MCPServers,
		}
		configJSON, _ := json.Marshal(mcpConfig)
		cmd = append(cmd, "--mcp-config", string(configJSON))
	} else if t.options.MCPServersPath != nil {
		cmd = append(cmd, "--mcp-config", *t.options.MCPServersPath)
	}

	// Add extra args
	for flag, value := range t.options.ExtraArgs {
		if value == nil {
			cmd = append(cmd, "--"+flag)
		} else {
			cmd = append(cmd, "--"+flag, *value)
		}
	}

	// Add prompt handling based on mode
	if t.isStreaming {
		cmd = append(cmd, "--input-format", "stream-json")
	} else {
		if prompt, ok := t.prompt.(string); ok {
			cmd = append(cmd, "--print", prompt)
		}
	}

	return cmd
}

// Connect starts the subprocess
func (t *SubprocessCLITransport) Connect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	// Create temp file for stderr
	stderrFile, err := os.CreateTemp("", "claude_stderr_*.log")
	if err != nil {
		return fmt.Errorf("failed to create stderr file: %w", err)
	}
	t.stderrFile = stderrFile

	// Build and start command
	cmdArgs := t.buildCommand()
	t.cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	t.cmd.Dir = t.cwd
	t.cmd.Env = append(os.Environ(), "CLAUDE_CODE_ENTRYPOINT=sdk-go")
	t.cmd.Stderr = stderrFile

	// Set up pipes
	if t.isStreaming {
		t.stdin, err = t.cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the process
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Claude Code: %w", err)
	}

	t.connected = true

	// Start reading stdout
	go t.readMessages()

	// Start streaming input if in streaming mode
	if t.isStreaming {
		go t.streamInput()
	}

	return nil
}

func (t *SubprocessCLITransport) streamInput() {
	defer func() {
		if t.closeStdinAfterPrompt && t.stdin != nil {
			t.stdin.Close()
		}
	}()

	switch prompt := t.prompt.(type) {
	case chan map[string]interface{}:
		for msg := range prompt {
			if t.stdin == nil {
				break
			}
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintln(t.stdin, string(data)); err != nil {
				break
			}
		}
	case <-chan map[string]interface{}:
		for msg := range prompt {
			if t.stdin == nil {
				break
			}
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintln(t.stdin, string(data)); err != nil {
				break
			}
		}
	}
}

func (t *SubprocessCLITransport) readMessages() {
	defer close(t.msgChan)
	defer close(t.doneChan)

	scanner := bufio.NewScanner(t.stdout)
	scanner.Buffer(make([]byte, maxBufferSize), maxBufferSize)

	jsonBuffer := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		jsonBuffer += line

		if len(jsonBuffer) > maxBufferSize {
			t.errChan <- fmt.Errorf("JSON message exceeded maximum buffer size")
			jsonBuffer = ""
			continue
		}

		var data MessageData
		if err := json.Unmarshal([]byte(jsonBuffer), &data); err == nil {
			jsonBuffer = ""
			
			// Skip control responses
			if data.Type == "control_response" {
				continue
			}

			select {
			case t.msgChan <- data:
			case <-t.doneChan:
				return
			}
		}
		// If JSON parsing fails, continue accumulating
	}

	if err := scanner.Err(); err != nil {
		t.errChan <- err
	}

	// Wait for process to complete
	if t.cmd != nil {
		if err := t.cmd.Wait(); err != nil {
			// Read stderr for error details
			if t.stderrFile != nil {
				t.stderrFile.Seek(0, 0)
				stderr, _ := io.ReadAll(t.stderrFile)
				if len(stderr) > 0 {
					t.errChan <- fmt.Errorf("command failed: %s", string(stderr))
				} else {
					t.errChan <- err
				}
			} else {
				t.errChan <- err
			}
		}
	}
}

// Disconnect terminates the subprocess
func (t *SubprocessCLITransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	// Close stdin if still open
	if t.stdin != nil {
		t.stdin.Close()
		t.stdin = nil
	}

	// Terminate process
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}

	// Clean up stderr file
	if t.stderrFile != nil {
		t.stderrFile.Close()
		os.Remove(t.stderrFile.Name())
		t.stderrFile = nil
	}

	t.connected = false

	return nil
}

// SendRequest sends additional messages in streaming mode
func (t *SubprocessCLITransport) SendRequest(messages []MessageData, metadata map[string]interface{}) error {
	if !t.isStreaming {
		return fmt.Errorf("SendRequest only works in streaming mode")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stdin == nil {
		return fmt.Errorf("stdin not available - stream may have ended")
	}

	for _, msg := range messages {
		// Ensure session_id is set
		if msg.SessionID == "" {
			if sessionID, ok := metadata["session_id"].(string); ok {
				msg.SessionID = sessionID
			} else {
				msg.SessionID = "default"
			}
		}

		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintln(t.stdin, string(data)); err != nil {
			return err
		}
	}

	return nil
}

// ReceiveMessages returns a channel of messages from the CLI
func (t *SubprocessCLITransport) ReceiveMessages() (<-chan MessageData, error) {
	if !t.connected {
		return nil, fmt.Errorf("not connected")
	}

	return t.msgChan, nil
}

// Interrupt sends an interrupt signal (not implemented for subprocess)
func (t *SubprocessCLITransport) Interrupt() error {
	// In Python, this sends SIGINT, but Go doesn't have a direct equivalent
	// You could implement this if needed
	return fmt.Errorf("interrupt not implemented for subprocess transport")
}