package main

import (
	"context"
	"fmt"

	sdk "claude-code-go-3sdk"
)

func basicExample() {
	fmt.Println("=== Basic Example ===")
	
	ctx := context.Background()
	
	for msg := range sdk.Query(ctx, "What is 2 + 2?", nil) {
		if assistantMsg, ok := msg.(*sdk.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(*sdk.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}
	fmt.Println()
}

func withOptionsExample() {
	fmt.Println("=== With Options Example ===")
	
	ctx := context.Background()
	
	options := &sdk.ClaudeCodeOptions{
		SystemPrompt: sdk.String("You are a helpful assistant that explains things simply."),
		MaxTurns:     sdk.Int(1),
	}
	
	for msg := range sdk.Query(ctx, "Explain what Go is in one sentence.", options) {
		if assistantMsg, ok := msg.(*sdk.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(*sdk.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}
	fmt.Println()
}

func withToolsExample() {
	fmt.Println("=== With Tools Example ===")
	
	ctx := context.Background()
	
	options := &sdk.ClaudeCodeOptions{
		AllowedTools: []string{"Read", "Write"},
		SystemPrompt: sdk.String("You are a helpful file assistant."),
	}
	
	for msg := range sdk.Query(ctx, "Create a file called hello.txt with 'Hello, World!' in it", options) {
		switch m := msg.(type) {
		case *sdk.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(*sdk.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *sdk.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}
	fmt.Println()
}

func main() {
	basicExample()
	withOptionsExample()
	withToolsExample()
}