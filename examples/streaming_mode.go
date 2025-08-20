package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	sdk "claude-code-go-3sdk"
)

func displayMessage(msg sdk.Message) {
	switch m := msg.(type) {
	case *sdk.UserMessage:
		// Handle string or blocks
		if str, ok := m.Content.(string); ok {
			fmt.Printf("User: %s\n", str)
		} else if blocks, ok := m.Content.([]sdk.ContentBlock); ok {
			for _, block := range blocks {
				if textBlock, ok := block.(*sdk.TextBlock); ok {
					fmt.Printf("User: %s\n", textBlock.Text)
				}
			}
		}
	case *sdk.AssistantMessage:
		for _, block := range m.Content {
			if textBlock, ok := block.(*sdk.TextBlock); ok {
				fmt.Printf("Claude: %s\n", textBlock.Text)
			}
		}
	case *sdk.ResultMessage:
		fmt.Println("Result ended")
		if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
			fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
		}
	}
}

func exampleBasicStreaming() {
	fmt.Println("=== Basic Streaming Example ===")

	ctx := context.Background()
	client := sdk.NewClient(nil)

	// Connect with empty stream for interactive use
	if err := client.Connect(ctx, nil); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	fmt.Println("User: What is 2+2?")
	if err := client.Query(ctx, "What is 2+2?", "default"); err != nil {
		log.Fatal(err)
	}

	// Receive complete response
	responseChan, err := client.ReceiveResponse(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range responseChan {
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleMultiTurnConversation() {
	fmt.Println("=== Multi-Turn Conversation Example ===")

	ctx := context.Background()
	client := sdk.NewClient(nil)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	// First turn
	fmt.Println("User: What's the capital of France?")
	if err := client.Query(ctx, "What's the capital of France?", "default"); err != nil {
		log.Fatal(err)
	}

	responseChan, err := client.ReceiveResponse(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range responseChan {
		displayMessage(msg)
	}

	// Second turn - follow-up
	fmt.Println("\nUser: What's the population of that city?")
	if err := client.Query(ctx, "What's the population of that city?", "default"); err != nil {
		log.Fatal(err)
	}

	responseChan, err = client.ReceiveResponse(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range responseChan {
		displayMessage(msg)
	}

	fmt.Println()
}

func exampleConcurrentResponses() {
	fmt.Println("=== Concurrent Send/Receive Example ===")

	ctx := context.Background()
	client := sdk.NewClient(nil)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	// Start receiving messages in background
	msgChan, err := client.ReceiveMessages(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Background goroutine to display messages
	go func() {
		for msg := range msgChan {
			displayMessage(msg)
		}
	}()

	// Send multiple messages with delays
	questions := []string{
		"What is 2 + 2?",
		"What is the square root of 144?",
		"What is 10% of 80?",
	}

	for _, question := range questions {
		fmt.Printf("\nUser: %s\n", question)
		if err := client.Query(ctx, question, "default"); err != nil {
			log.Fatal(err)
		}
		time.Sleep(3 * time.Second)
	}

	// Give time for final responses
	time.Sleep(2 * time.Second)

	fmt.Println()
}

func exampleWithOptions() {
	fmt.Println("=== Custom Options Example ===")

	ctx := context.Background()

	// Configure options
	options := &sdk.ClaudeCodeOptions{
		AllowedTools:      []string{"Read", "Write"},
		MaxThinkingTokens: 10000,
		SystemPrompt:      sdk.String("You are a helpful coding assistant."),
	}

	client := sdk.NewClient(options)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	fmt.Println("User: Create a simple hello.txt file with a greeting message")
	if err := client.Query(ctx, "Create a simple hello.txt file with a greeting message", "default"); err != nil {
		log.Fatal(err)
	}

	toolUses := []string{}
	responseChan, err := client.ReceiveResponse(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range responseChan {
		if assistantMsg, ok := msg.(*sdk.AssistantMessage); ok {
			displayMessage(msg)
			for _, block := range assistantMsg.Content {
				if toolBlock, ok := block.(*sdk.ToolUseBlock); ok {
					toolUses = append(toolUses, toolBlock.Name)
				}
			}
		} else {
			displayMessage(msg)
		}
	}

	if len(toolUses) > 0 {
		fmt.Printf("Tools used: %v\n", toolUses)
	}

	fmt.Println()
}

func exampleManualMessageHandling() {
	fmt.Println("=== Manual Message Handling Example ===")

	ctx := context.Background()
	client := sdk.NewClient(nil)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	if err := client.Query(ctx, "List 5 programming languages and their main use cases", "default"); err != nil {
		log.Fatal(err)
	}

	// Manually process messages with custom logic
	languagesFound := []string{}
	languages := []string{"Python", "JavaScript", "Java", "C++", "Go", "Rust", "Ruby"}

	msgChan, err := client.ReceiveMessages(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range msgChan {
		if assistantMsg, ok := msg.(*sdk.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(*sdk.TextBlock); ok {
					text := textBlock.Text
					fmt.Printf("Claude: %s\n", text)
					
					// Custom logic: extract language names
					for _, lang := range languages {
						found := false
						for _, foundLang := range languagesFound {
							if foundLang == lang {
								found = true
								break
							}
						}
						if !found && contains(text, lang) {
							languagesFound = append(languagesFound, lang)
							fmt.Printf("Found language: %s\n", lang)
						}
					}
				}
			}
		} else if _, ok := msg.(*sdk.ResultMessage); ok {
			displayMessage(msg)
			fmt.Printf("Total languages mentioned: %d\n", len(languagesFound))
			break
		}
	}

	fmt.Println()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
	       len(s) > len(substr) && contains(s[1:], substr)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run streaming_mode.go <example_name>")
		fmt.Println("\nAvailable examples:")
		fmt.Println("  all - Run all examples")
		fmt.Println("  basic_streaming")
		fmt.Println("  multi_turn_conversation")
		fmt.Println("  concurrent_responses")
		fmt.Println("  with_options")
		fmt.Println("  manual_message_handling")
		os.Exit(0)
	}

	examples := map[string]func(){
		"basic_streaming":         exampleBasicStreaming,
		"multi_turn_conversation": exampleMultiTurnConversation,
		"concurrent_responses":    exampleConcurrentResponses,
		"with_options":            exampleWithOptions,
		"manual_message_handling": exampleManualMessageHandling,
	}

	exampleName := os.Args[1]

	if exampleName == "all" {
		for _, example := range examples {
			example()
			fmt.Println(strings.Repeat("-", 50) + "\n")
		}
	} else if example, ok := examples[exampleName]; ok {
		example()
	} else {
		fmt.Printf("Error: Unknown example '%s'\n", exampleName)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  all - Run all examples")
		for name := range examples {
			fmt.Printf("  %s\n", name)
		}
		os.Exit(1)
	}
}