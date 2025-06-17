package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcphost/internal/agent"
	"github.com/mark3labs/mcphost/internal/config"
	"github.com/mark3labs/mcphost/internal/models"
	"github.com/mark3labs/mcphost/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var scriptCmd = &cobra.Command{
	Use:   "script <script-file>",
	Short: "Execute a script file with YAML frontmatter configuration",
	Long: `Execute a script file that contains YAML frontmatter with configuration
and a prompt. The script file can contain MCP server configurations,
model settings, and other options.

Example script file:
---
model: "anthropic:claude-sonnet-4-20250514"
max-steps: 10
mcp-servers:
  filesystem:
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
---
List the files in ${directory} and tell me about them.

The script command supports the same flags as the main command,
which will override any settings in the script file.

Variable substitution:
Variables in the script can be substituted using ${variable} syntax.
Pass variables using --args:variable value syntax:

  mcphost script myscript.sh --args:directory /tmp --args:name "John"

This will replace ${directory} with "/tmp" and ${name} with "John" in the script.`,
	Args: cobra.ExactArgs(1),
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true, // Allow unknown flags for variable substitution
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		// Override config with frontmatter values from the script file
		scriptFile := args[0]
		variables := parseCustomVariables(cmd)
		overrideConfigWithFrontmatter(scriptFile, variables, cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptFile := args[0]

		// Parse custom variables from unknown flags
		variables := parseCustomVariables(cmd)

		return runScriptCommand(context.Background(), scriptFile, variables, cmd)
	},
}

func init() {
	rootCmd.AddCommand(scriptCmd)

	// Add the same flags as the root command, but they will override script settings
	scriptCmd.Flags().StringVar(&systemPromptFile, "system-prompt", "", "system prompt text or path to text file")
	scriptCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "model to use (format: provider:model)")
	scriptCmd.Flags().BoolVar(&debugMode, "debug", false, "enable debug logging")
	scriptCmd.Flags().StringVarP(&promptFlag, "prompt", "p", "", "override the prompt from the script file")
	scriptCmd.Flags().BoolVar(&quietFlag, "quiet", false, "suppress all output")
	scriptCmd.Flags().BoolVar(&noExitFlag, "no-exit", false, "prevent script from exiting, show input prompt instead")
	scriptCmd.Flags().IntVar(&maxSteps, "max-steps", 0, "maximum number of agent steps (0 for unlimited)")
	scriptCmd.Flags().StringVar(&providerURL, "provider-url", "", "base URL for the provider API (applies to OpenAI, Anthropic, Ollama, and Google)")
	scriptCmd.Flags().StringVar(&providerAPIKey, "provider-api-key", "", "API key for the provider (applies to OpenAI, Anthropic, and Google)")

	// Model generation parameters
	scriptCmd.Flags().IntVar(&maxTokens, "max-tokens", 4096, "maximum number of tokens in the response")
	scriptCmd.Flags().Float32Var(&temperature, "temperature", 0.7, "controls randomness in responses (0.0-1.0)")
	scriptCmd.Flags().Float32Var(&topP, "top-p", 0.95, "controls diversity via nucleus sampling (0.0-1.0)")
	scriptCmd.Flags().Int32Var(&topK, "top-k", 40, "controls diversity by limiting top K tokens to sample from")
	scriptCmd.Flags().StringSliceVar(&stopSequences, "stop-sequences", nil, "custom stop sequences (comma-separated)")

	// Bind script command flags to viper so they have proper precedence
	viper.BindPFlag("system-prompt", scriptCmd.Flags().Lookup("system-prompt"))
	viper.BindPFlag("model", scriptCmd.Flags().Lookup("model"))
	viper.BindPFlag("debug", scriptCmd.Flags().Lookup("debug"))
	viper.BindPFlag("max-steps", scriptCmd.Flags().Lookup("max-steps"))
	viper.BindPFlag("provider-url", scriptCmd.Flags().Lookup("provider-url"))
	viper.BindPFlag("provider-api-key", scriptCmd.Flags().Lookup("provider-api-key"))
	viper.BindPFlag("max-tokens", scriptCmd.Flags().Lookup("max-tokens"))
	viper.BindPFlag("temperature", scriptCmd.Flags().Lookup("temperature"))
	viper.BindPFlag("top-p", scriptCmd.Flags().Lookup("top-p"))
	viper.BindPFlag("top-k", scriptCmd.Flags().Lookup("top-k"))
	viper.BindPFlag("stop-sequences", scriptCmd.Flags().Lookup("stop-sequences"))
}

// overrideConfigWithFrontmatter parses the script file and overrides viper config with frontmatter values
// This is the only purpose of this function - to apply frontmatter configuration to viper
func overrideConfigWithFrontmatter(scriptFile string, variables map[string]string, cmd *cobra.Command) {
	// Parse the script file to get frontmatter configuration
	scriptConfig, err := parseScriptFile(scriptFile, variables)
	if err != nil {
		// If we can't parse the script file, just continue with existing config
		// The error will be handled again in runScriptCommand
		return
	}

	// Override viper values with frontmatter values (only if flags weren't explicitly set)
	if scriptConfig.Model != "" && !cmd.Flags().Changed("model") {
		viper.Set("model", scriptConfig.Model)
	}
	if scriptConfig.MaxSteps != 0 && !cmd.Flags().Changed("max-steps") {
		viper.Set("max-steps", scriptConfig.MaxSteps)
	}
	if scriptConfig.Debug && !cmd.Flags().Changed("debug") {
		viper.Set("debug", scriptConfig.Debug)
	}
	if scriptConfig.SystemPrompt != "" && !cmd.Flags().Changed("system-prompt") {
		viper.Set("system-prompt", scriptConfig.SystemPrompt)
	}
	if scriptConfig.ProviderAPIKey != "" && !cmd.Flags().Changed("provider-api-key") {
		viper.Set("provider-api-key", scriptConfig.ProviderAPIKey)
	}
	if scriptConfig.ProviderURL != "" && !cmd.Flags().Changed("provider-url") {
		viper.Set("provider-url", scriptConfig.ProviderURL)
	}
	if scriptConfig.MaxTokens != 0 && !cmd.Flags().Changed("max-tokens") {
		viper.Set("max-tokens", scriptConfig.MaxTokens)
	}
	if scriptConfig.Temperature != nil && !cmd.Flags().Changed("temperature") {
		viper.Set("temperature", *scriptConfig.Temperature)
	}
	if scriptConfig.TopP != nil && !cmd.Flags().Changed("top-p") {
		viper.Set("top-p", *scriptConfig.TopP)
	}
	if scriptConfig.TopK != nil && !cmd.Flags().Changed("top-k") {
		viper.Set("top-k", *scriptConfig.TopK)
	}
	if len(scriptConfig.StopSequences) > 0 && !cmd.Flags().Changed("stop-sequences") {
		viper.Set("stop-sequences", scriptConfig.StopSequences)
	}
	if scriptConfig.NoExit && !cmd.Flags().Changed("no-exit") {
		// Set the global noExitFlag variable if it wasn't explicitly set via command line
		noExitFlag = scriptConfig.NoExit
	}
}

// parseCustomVariables extracts custom variables from command line arguments
func parseCustomVariables(_ *cobra.Command) map[string]string {
	variables := make(map[string]string)

	// Get all arguments passed to the command
	args := os.Args[1:] // Skip program name

	// Find the script subcommand position
	scriptPos := -1
	for i, arg := range args {
		if arg == "script" {
			scriptPos = i
			break
		}
	}

	if scriptPos == -1 {
		return variables
	}

	// Parse arguments after the script file
	scriptFileFound := false

	for i := scriptPos + 1; i < len(args); i++ {
		arg := args[i]

		// Skip the script file argument (first non-flag after "script")
		if !scriptFileFound && !strings.HasPrefix(arg, "-") {
			scriptFileFound = true
			continue
		}

		// Parse custom variables with --args: prefix
		if strings.HasPrefix(arg, "--args:") {
			varName := strings.TrimPrefix(arg, "--args:")
			if varName == "" {
				continue // Skip malformed --args: without name
			}

			// Check if we have a value
			if i+1 < len(args) {
				varValue := args[i+1]

				// Make sure the next arg isn't a flag
				if !strings.HasPrefix(varValue, "-") {
					variables[varName] = varValue
					i++ // Skip the value
				} else {
					// No value provided, treat as empty string
					variables[varName] = ""
				}
			} else {
				// No value provided, treat as empty string
				variables[varName] = ""
			}
		}
	}

	return variables
}

func runScriptCommand(ctx context.Context, scriptFile string, variables map[string]string, _ *cobra.Command) error {
	// Parse the script file to get MCP servers and prompt
	scriptConfig, err := parseScriptFile(scriptFile, variables)
	if err != nil {
		return fmt.Errorf("failed to parse script file: %v", err)
	}

	// Get MCP config - use script servers if available, otherwise use global viper config
	var mcpConfig *config.Config
	if len(scriptConfig.MCPServers) > 0 {
		// Use MCP servers from script, but get other config values from viper
		mcpConfig = &config.Config{
			MCPServers: scriptConfig.MCPServers,
		}
		if err := viper.Unmarshal(mcpConfig); err != nil {
			return fmt.Errorf("failed to unmarshal config: %v", err)
		}
		// Restore the MCP servers from script (viper.Unmarshal might have overwritten them)
		mcpConfig.MCPServers = scriptConfig.MCPServers
	} else {
		// Get MCP config from the global viper instance (already loaded by initConfig)
		mcpConfig = &config.Config{
			MCPServers: make(map[string]config.MCPServerConfig),
		}
		if err := viper.Unmarshal(mcpConfig); err != nil {
			return fmt.Errorf("failed to unmarshal MCP config: %v", err)
		}
	}

	// Validate the config
	if err := mcpConfig.Validate(); err != nil {
		return fmt.Errorf("invalid MCP config: %v", err)
	}

	// Get final prompt - prioritize command line flag, then script content
	finalPrompt := viper.GetString("prompt")
	if finalPrompt == "" && scriptConfig.Prompt != "" {
		finalPrompt = scriptConfig.Prompt
	}

	// Get final no-exit setting - prioritize command line flag, then script config
	finalNoExit := noExitFlag || scriptConfig.NoExit

	// Validate that --no-exit is only used when there's a prompt
	if finalNoExit && finalPrompt == "" {
		return fmt.Errorf("--no-exit flag can only be used when there's a prompt (either from script content or --prompt flag)")
	}

	// Run the script using the unified agentic loop
	return runScriptMode(ctx, mcpConfig, finalPrompt, finalNoExit)
}

// mergeScriptConfig and setScriptValuesInViper functions removed
// Configuration override is now handled in overrideConfigWithFrontmatter in the PreRun hook

// parseScriptFile parses a script file with YAML frontmatter and returns config
func parseScriptFile(filename string, variables map[string]string) (*config.Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip shebang line if present
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#!") {
			// If it's not a shebang, we need to process this line
			return parseScriptContent(line+"\n"+readRemainingLines(scanner), variables)
		}
	}

	// Read the rest of the file
	content := readRemainingLines(scanner)
	return parseScriptContent(content, variables)
}

// readRemainingLines reads all remaining lines from a scanner
func readRemainingLines(scanner *bufio.Scanner) string {
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n")
}

// parseScriptContent parses the content to extract YAML frontmatter and prompt
func parseScriptContent(content string, variables map[string]string) (*config.Config, error) {
	// First, validate that all declared variables are provided
	if err := validateVariables(content, variables); err != nil {
		return nil, err
	}

	// Substitute variables in the content
	content = substituteVariables(content, variables)

	lines := strings.Split(content, "\n")

	// Find YAML frontmatter between --- delimiters
	var yamlLines []string
	var promptLines []string
	var inFrontmatter bool
	var foundFrontmatter bool
	var frontmatterEnd int = -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comment lines (lines starting with #)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for frontmatter start
		if trimmed == "---" && !inFrontmatter {
			// Start of frontmatter
			inFrontmatter = true
			foundFrontmatter = true
			continue
		}

		// Check for frontmatter end
		if trimmed == "---" && inFrontmatter {
			// End of frontmatter
			inFrontmatter = false
			frontmatterEnd = i + 1
			continue
		}

		// Collect frontmatter lines
		if inFrontmatter {
			yamlLines = append(yamlLines, line)
		}
	}

	// Extract prompt (everything after frontmatter)
	if foundFrontmatter && frontmatterEnd != -1 && frontmatterEnd < len(lines) {
		promptLines = lines[frontmatterEnd:]
	} else if !foundFrontmatter {
		// If no frontmatter found, treat entire content as prompt
		promptLines = lines
		yamlLines = []string{} // Empty YAML
	}

	// Parse YAML frontmatter
	var scriptConfig config.Config
	if len(yamlLines) > 0 {
		yamlContent := strings.Join(yamlLines, "\n")
		if err := yaml.Unmarshal([]byte(yamlContent), &scriptConfig); err != nil {
			return nil, fmt.Errorf("failed to parse YAML frontmatter: %v\nYAML content:\n%s", err, yamlContent)
		}
	}

	// Set prompt from content after frontmatter
	if len(promptLines) > 0 {
		prompt := strings.Join(promptLines, "\n")
		prompt = strings.TrimSpace(prompt) // Remove leading/trailing whitespace
		if prompt != "" {
			scriptConfig.Prompt = prompt
		}
	}

	return &scriptConfig, nil
}

// findVariables extracts all unique variable names from ${variable} patterns in content
func findVariables(content string) []string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(content, -1)

	seenVars := make(map[string]bool)
	var variables []string

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !seenVars[varName] {
				seenVars[varName] = true
				variables = append(variables, varName)
			}
		}
	}

	return variables
}

// validateVariables checks that all declared variables in the content are provided
func validateVariables(content string, variables map[string]string) error {
	declaredVars := findVariables(content)

	var missingVars []string
	for _, varName := range declaredVars {
		if _, exists := variables[varName]; !exists {
			missingVars = append(missingVars, varName)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required variables: %s\nProvide them using --args:variable value syntax", strings.Join(missingVars, ", "))
	}

	return nil
}

// substituteVariables replaces ${variable} patterns with their values
func substituteVariables(content string, variables map[string]string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]

		// Look up the variable value
		if value, exists := variables[varName]; exists {
			return value
		}

		// If variable not found, leave it as is
		return match
	})
}

// runScriptMode executes the script using the unified agentic loop
func runScriptMode(ctx context.Context, mcpConfig *config.Config, prompt string, noExit bool) error {
	// Set up logging
	if debugMode || mcpConfig.Debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Get final values from viper and script config
	finalModel := viper.GetString("model")
	if finalModel == "" && mcpConfig.Model != "" {
		finalModel = mcpConfig.Model
	}
	if finalModel == "" {
		finalModel = "anthropic:claude-sonnet-4-20250514" // default
	}

	finalSystemPrompt := viper.GetString("system-prompt")
	if finalSystemPrompt == "" && mcpConfig.SystemPrompt != "" {
		finalSystemPrompt = mcpConfig.SystemPrompt
	}

	finalDebug := viper.GetBool("debug") || mcpConfig.Debug
	finalMaxSteps := viper.GetInt("max-steps")
	if finalMaxSteps == 0 && mcpConfig.MaxSteps != 0 {
		finalMaxSteps = mcpConfig.MaxSteps
	}

	finalProviderURL := viper.GetString("provider-url")
	if finalProviderURL == "" && mcpConfig.ProviderURL != "" {
		finalProviderURL = mcpConfig.ProviderURL
	}

	finalProviderAPIKey := viper.GetString("provider-api-key")
	if finalProviderAPIKey == "" && mcpConfig.ProviderAPIKey != "" {
		finalProviderAPIKey = mcpConfig.ProviderAPIKey
	}

	finalMaxTokens := viper.GetInt("max-tokens")
	if finalMaxTokens == 0 && mcpConfig.MaxTokens != 0 {
		finalMaxTokens = mcpConfig.MaxTokens
	}
	if finalMaxTokens == 0 {
		finalMaxTokens = 4096 // default
	}

	finalTemperature := float32(viper.GetFloat64("temperature"))
	if finalTemperature == 0 && mcpConfig.Temperature != nil {
		finalTemperature = *mcpConfig.Temperature
	}
	if finalTemperature == 0 {
		finalTemperature = 0.7 // default
	}

	finalTopP := float32(viper.GetFloat64("top-p"))
	if finalTopP == 0 && mcpConfig.TopP != nil {
		finalTopP = *mcpConfig.TopP
	}
	if finalTopP == 0 {
		finalTopP = 0.95 // default
	}

	finalTopK := int32(viper.GetInt("top-k"))
	if finalTopK == 0 && mcpConfig.TopK != nil {
		finalTopK = *mcpConfig.TopK
	}
	if finalTopK == 0 {
		finalTopK = 40 // default
	}

	finalStopSequences := viper.GetStringSlice("stop-sequences")
	if len(finalStopSequences) == 0 && len(mcpConfig.StopSequences) > 0 {
		finalStopSequences = mcpConfig.StopSequences
	}

	// Load system prompt
	systemPrompt, err := config.LoadSystemPrompt(finalSystemPrompt)
	if err != nil {
		return fmt.Errorf("failed to load system prompt: %v", err)
	}

	// Create model configuration
	modelConfig := &models.ProviderConfig{
		ModelString:    finalModel,
		SystemPrompt:   systemPrompt,
		ProviderAPIKey: finalProviderAPIKey,
		ProviderURL:    finalProviderURL,
		MaxTokens:      finalMaxTokens,
		Temperature:    &finalTemperature,
		TopP:           &finalTopP,
		TopK:           &finalTopK,
		StopSequences:  finalStopSequences,
	}

	// Create agent configuration
	agentConfig := &agent.AgentConfig{
		ModelConfig:  modelConfig,
		MCPConfig:    mcpConfig,
		SystemPrompt: systemPrompt,
		MaxSteps:     finalMaxSteps,
	}

	// Create the agent
	mcpAgent, err := agent.NewAgent(ctx, agentConfig)
	if err != nil {
		return fmt.Errorf("failed to create agent: %v", err)
	}
	defer mcpAgent.Close()

	// Get model name for display
	parts := strings.SplitN(finalModel, ":", 2)
	modelName := "Unknown"
	if len(parts) == 2 {
		modelName = parts[1]
	}

	// Create CLI interface (skip if quiet mode)
	var cli *ui.CLI
	if !quietFlag {
		cli, err = ui.NewCLI(finalDebug)
		if err != nil {
			return fmt.Errorf("failed to create CLI: %v", err)
		}

		// Log successful initialization
		if len(parts) == 2 {
			cli.DisplayInfo(fmt.Sprintf("Model loaded: %s (%s)", parts[0], parts[1]))
		}

		tools := mcpAgent.GetTools()
		cli.DisplayInfo(fmt.Sprintf("Loaded %d tools from MCP servers", len(tools)))

		// Display debug configuration if debug mode is enabled
		if finalDebug {
			debugConfig := map[string]any{
				"model":         finalModel,
				"max-steps":     finalMaxSteps,
				"max-tokens":    finalMaxTokens,
				"temperature":   finalTemperature,
				"top-p":         finalTopP,
				"top-k":         finalTopK,
				"provider-url":  finalProviderURL,
				"system-prompt": finalSystemPrompt,
			}

			// Only include non-empty stop sequences
			if len(finalStopSequences) > 0 {
				debugConfig["stop-sequences"] = finalStopSequences
			}

			// Only include API keys if they're set (but don't show the actual values for security)
			if finalProviderAPIKey != "" {
				debugConfig["provider-api-key"] = "[SET]"
			}

			cli.DisplayDebugConfig(debugConfig)
		}
	}

	// Prepare data for slash commands
	var serverNames []string
	for name := range mcpConfig.MCPServers {
		serverNames = append(serverNames, name)
	}

	tools := mcpAgent.GetTools()
	var toolNames []string
	for _, tool := range tools {
		if info, err := tool.Info(ctx); err == nil {
			toolNames = append(toolNames, info.Name)
		}
	}

	// Configure and run unified agentic loop
	var messages []*schema.Message
	config := AgenticLoopConfig{
		IsInteractive:    prompt == "", // If no prompt, start in interactive mode
		InitialPrompt:    prompt,
		ContinueAfterRun: noExit,
		Quiet:            quietFlag,
		ServerNames:      serverNames,
		ToolNames:        toolNames,
		ModelName:        modelName,
		MCPConfig:        mcpConfig,
	}

	return runAgenticLoop(ctx, mcpAgent, cli, messages, config)
}
