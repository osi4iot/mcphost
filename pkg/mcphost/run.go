package mcphost

import (
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/nats-io/nats.go"
	"github.com/osi4iot/mcphost/internal/agent"
	"github.com/osi4iot/mcphost/internal/config"
	"github.com/osi4iot/mcphost/internal/models"
	"github.com/osi4iot/mcphost/internal/tokens"
)

func (h *mcpHost) RunMCPHost() error {
	// Initialize token counters
	tokens.InitializeTokenCounters()

	mcpConfig := &config.Config{
		MCPServers:     h.config.MCPServers,
		Model:          h.config.Model,
		MaxSteps:       h.config.MaxSteps,
		Debug:          h.config.Debug,
		Compact:        false,
		SystemPrompt:   h.config.SystemPrompt,
		ProviderAPIKey: h.config.ProviderAPIKey,
		ProviderURL:    h.config.ProviderURL,
		Prompt:         "",
		NoExit:         false,
		Stream:         nil, // Use default streaming behavior
		MaxTokens:      h.config.MaxTokens,
		Temperature:    h.config.Temperature,
		TopP:           h.config.TopP,
		TopK:           h.config.TopK,
		StopSequences:  []string{},
		TLSSkipVerify:  false,
	}
	var err error

	systemPrompt := h.config.SystemPrompt
	temperature := *h.config.Temperature
	topP := *h.config.TopP
	var topK int32 = 40
	var numGPU int32 = -1
	var mainGPU int32 = 0
	providerAPIKey := h.config.ProviderAPIKey
	providerURL := h.config.ProviderURL
	modelString := h.config.Model
	if modelString == "" {
		modelString = "openai:gpt-4o-mini"
	}

	maxTokens := h.config.MaxTokens
	stopSequences := []string{}
	maxSteps := h.config.MaxSteps
	streamingEnabled := false

	modelConfig := &models.ProviderConfig{
		ModelString:    modelString,
		SystemPrompt:   systemPrompt,
		ProviderAPIKey: providerAPIKey,
		ProviderURL:    providerURL,
		MaxTokens:      maxTokens,
		Temperature:    &temperature,
		TopP:           &topP,
		TopK:           &topK,
		StopSequences:  stopSequences,
		NumGPU:         &numGPU,
		MainGPU:        &mainGPU,
	}

	// Create the agent using the factory
	mcpAgent, err := agent.CreateAgent(h.ctx, &agent.AgentCreationOptions{
		ModelConfig:      modelConfig,
		MCPConfig:        mcpConfig,
		SystemPrompt:     systemPrompt,
		MaxSteps:         maxSteps,
		StreamingEnabled: streamingEnabled,
		ShowSpinner:      false,
		Quiet:            false,
		SpinnerFunc:      nil, // Use default spinner function
	})
	if err != nil {
		return fmt.Errorf("failed to create agent: %v", err)
	}
	defer mcpAgent.Close()

	// Prepare data for slash commands
	var serverNames []string
	for name := range mcpConfig.MCPServers {
		serverNames = append(serverNames, name)
	}
	h.serverNames = serverNames

	tools := mcpAgent.GetTools()
	var toolNames []string
	for _, tool := range tools {
		if info, err := tool.Info(h.ctx); err == nil {
			toolNames = append(toolNames, info.Name)
		}
	}
	h.toolNames = toolNames

	return h.runInteractiveLoop(mcpAgent)
}

func (h *mcpHost) runInteractiveLoop(mcpAgent *agent.Agent) error {
	for {
		select {
		case <-h.ctx.Done():
			fmt.Println("Context cancelled, stopping runInteractiveLoop")
			return nil
		case prompt, ok := <-h.config.inputChan:
			if !ok {
				return fmt.Errorf("input channel closed, stopping runInteractiveLoop")
			}

			h.mu.RLock()
			tempMessages := append(*h.messages, schema.UserMessage(prompt))
			h.mu.RUnlock()

			// Process the user input with tool calls
			message, conversationMessages, err := h.runAgenticStep(mcpAgent, tempMessages)
			if err != nil {
				fmt.Printf("Error processing user input: %v\n", err)
			}

			h.config.outputChan <- message.Content

			h.mu.Lock()
			*h.messages = append(*h.messages, conversationMessages...)
			h.mu.Unlock()
		}
	}
}

// runAgenticStep processes a single step of the agentic loop (handles tool calls)
func (h *mcpHost) runAgenticStep(mcpAgent *agent.Agent, messages []*schema.Message) (*schema.Message, []*schema.Message, error) {
	result, err := mcpAgent.GenerateWithLoopAndStreaming(h.ctx, messages,
		// Tool call handler - called when a tool is about to be executed
		func(toolName, toolArgs string) {
			if h.config.Debug {
				fmt.Printf("Tool call: %s with args: %s\n", toolName, toolArgs)
			}
		},
		// Tool execution handler - called when tool execution starts/ends
		func(toolName string, isStarting bool) {
			if h.config.Debug {
				if isStarting {
					fmt.Printf("Starting tool: %s\n", toolName)
				} else {
					fmt.Printf("Finished tool: %s\n", toolName)
				}
			}
		},
		// Tool result handler - called when a tool execution completes
		func(toolName, toolArgs, result string, isError bool) {
			resultContent := result

			// Try to parse as MCP content structure
			var mcpContent struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			}

			// First try to unmarshal as-is
			if err := json.Unmarshal([]byte(result), &mcpContent); err == nil {
				// Extract text from MCP content structure
				if len(mcpContent.Content) > 0 && mcpContent.Content[0].Type == "text" {
					resultContent = mcpContent.Content[0].Text
				}
			} else {
				// If that fails, try unquoting first (in case it's double-encoded)
				var unquoted string
				if err := json.Unmarshal([]byte(result), &unquoted); err == nil {
					if err := json.Unmarshal([]byte(unquoted), &mcpContent); err == nil {
						if len(mcpContent.Content) > 0 && mcpContent.Content[0].Type == "text" {
							resultContent = mcpContent.Content[0].Text
						}
					}
				}
			}

			if isError {
				fmt.Printf("Tool error for %s: %s\n", toolName, resultContent)
			}

			if h.config.Debug {
				fmt.Printf("Tool result for %s: %s\n", toolName, resultContent)
			}
		},
		// Response handler - called when the LLM generates a response
		func(content string) {
			fmt.Printf("LLM response: %s\n", content)
		},
		// Tool call content handler - called when content accompanies tool calls
		func(content string) {},
		nil, // Add streaming callback as the last parameter
	)

	if err != nil {
		return nil, nil, err
	}

	// Get the final response and conversation messages
	response := result.FinalResponse
	conversationMessages := result.ConversationMessages

	// Return the final response and all conversation messages
	return response, conversationMessages, nil
}

func (h *mcpHost) RunNatsSubscription(mcpAgent *agent.Agent) error {
	// Main interaction logic
	errChan := make(chan error, 1)

	subjectIn := ""
	sub, err := h.config.NatsClient.Subscribe(subjectIn, func(m *nats.Msg) {
		// Get user input
		prompt := string(m.Data)
		if prompt == "" {
			return // Skip empty prompts
		}
		fmt.Printf("Received user input: %s\n", prompt)

		// Agregar el mensaje del usuario
		h.mu.RLock()
		tempMessages := append(*h.messages, schema.UserMessage(prompt))
		h.mu.RUnlock()

		// Process the user input with tool calls
		_, conversationMessages, err := h.runNatsAgenticStep(mcpAgent, tempMessages)
		if err != nil {
			fmt.Printf("Error processing user input: %v\n", err)
			select {
			case errChan <- err:
			default:
			}
			return
		}

		h.mu.Lock()
		*h.messages = append(*h.messages, conversationMessages...)
		h.mu.Unlock()
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS subject: %v", err)
	}

	fmt.Printf("Subscribed to NATS subject: %s\n", subjectIn)

	// **PUNTO CLAVE**: Esperar hasta que el contexto se cancele
	select {
	case <-h.ctx.Done():
		fmt.Println("Context cancelled, unsubscribing...")
		sub.Unsubscribe()
		return h.ctx.Err()
	case err := <-errChan:
		fmt.Printf("Critical error received: %v\n", err)
		sub.Unsubscribe()
		return err
	}
}


// runNatsAgenticStep processes a single step of the agentic loop (handles tool calls)
func (h *mcpHost) runNatsAgenticStep(mcpAgent *agent.Agent, messages []*schema.Message) (*schema.Message, []*schema.Message, error) {
	result, err := mcpAgent.GenerateWithLoopAndStreaming(h.ctx, messages,
		// Tool call handler - called when a tool is about to be executed
		func(toolName, toolArgs string) {
			if h.config.Debug {
				fmt.Printf("Tool call: %s with args: %s\n", toolName, toolArgs)
			}
		},
		// Tool execution handler - called when tool execution starts/ends
		func(toolName string, isStarting bool) {
			if h.config.Debug {
				if isStarting {
					fmt.Printf("Starting tool: %s\n", toolName)
				} else {
					fmt.Printf("Finished tool: %s\n", toolName)
				}
			}
		},
		// Tool result handler - called when a tool execution completes
		func(toolName, toolArgs, result string, isError bool) {
			resultContent := result

			// Try to parse as MCP content structure
			var mcpContent struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			}

			// First try to unmarshal as-is
			if err := json.Unmarshal([]byte(result), &mcpContent); err == nil {
				// Extract text from MCP content structure
				if len(mcpContent.Content) > 0 && mcpContent.Content[0].Type == "text" {
					resultContent = mcpContent.Content[0].Text
				}
			} else {
				// If that fails, try unquoting first (in case it's double-encoded)
				var unquoted string
				if err := json.Unmarshal([]byte(result), &unquoted); err == nil {
					if err := json.Unmarshal([]byte(unquoted), &mcpContent); err == nil {
						if len(mcpContent.Content) > 0 && mcpContent.Content[0].Type == "text" {
							resultContent = mcpContent.Content[0].Text
						}
					}
				}
			}

			if isError {
				fmt.Printf("Tool error for %s: %s\n", toolName, resultContent)
			}

			if h.config.Debug {
				fmt.Printf("Tool result for %s: %s\n", toolName, resultContent)
			}
		},
		// Response handler - called when the LLM generates a response
		func(content string) {
			// fmt.Printf("LLM response: %s\n", content)
			subjectOut := ""
			message := []byte(content)
			if err := h.config.NatsClient.Publish(subjectOut, message); err != nil {
				fmt.Println("Error publishing to NATS:", err)
			}
		},
		// Tool call content handler - called when content accompanies tool calls
		func(content string) {},
		nil, // Add streaming callback as the last parameter
	)

	if err != nil {
		return nil, nil, err
	}

	// Get the final response and conversation messages
	response := result.FinalResponse
	conversationMessages := result.ConversationMessages

	// Return the final response and all conversation messages
	return response, conversationMessages, nil
}
