package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// AgentGenerator Agent 项目生成器
type AgentGenerator struct {
	projectName string
	outputDir   string
}

// NewAgentGenerator creates new agent generator
func NewAgentGenerator(projectName, outputDir string) *AgentGenerator {
	return &AgentGenerator{
		projectName: projectName,
		outputDir:   outputDir,
	}
}

// Generate generates agent project
func (g *AgentGenerator) Generate() error {
	dirs := []string{
		"internal/agent",
		"internal/llm",
		"internal/memory",
		"internal/handler",
		"prompts",
		"configs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"main.go":                   g.mainContent(),
		"internal/agent/agent.go":   g.agentContent(),
		"internal/llm/client.go":    g.llmClientContent(),
		"internal/memory/memory.go": g.memoryContent(),
		"internal/handler/http.go":  g.handlerContent(),
		"prompts/system.txt":        g.promptContent(),
		"configs/config.yaml":       g.configContent(),
		"go.mod":                    g.goModContent(),
		"readme.md":                 g.readmeContent(),
	}

	for path, content := range files {
		fullPath := filepath.Join(g.outputDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (g *AgentGenerator) mainContent() string {
	return fmt.Sprintf(`package main

import (
	"log"
	"%s/internal/agent"
	"%s/internal/llm"
	"%s/internal/handler"
)

func main() {
	// Initialize LLM client
	llmClient := llm.NewClient()
	
	// Create agent
	agt := agent.NewAgent(llmClient)
	
	// Start HTTP server
	log.Println("Starting agent server on :8080")
	handler.Start(agt)
}
`, g.projectName, g.projectName, g.projectName)
}

func (g *AgentGenerator) agentContent() string {
	return `package agent

import (
	"context"
	"strings"
)

// Agent AI agent
type Agent struct {
	llm LLMClient
}

// LLMClient LLM interface
type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// NewAgent creates new agent
func NewAgent(llm LLMClient) *Agent {
	return &Agent{llm: llm}
}

// Run runs agent with input
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	prompt := buildPrompt(input)
	response, err := a.llm.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response), nil
}

func buildPrompt(input string) string {
	return "You are a helpful assistant.\n\nUser: " + input + "\n\nAssistant:"
}
`
}

func (g *AgentGenerator) llmClientContent() string {
	return `package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	apiKey  string
	model  string
	baseURL string
}

func NewClient() *Client {
	return &Client{
		apiKey:  os.Getenv("DMXAPI_KEY"),
		model:  "qwen3.5-plus-free",
		baseURL: "https://www.dmxapi.cn",
	}
}

type Message struct {
	Role    string "json:\"role\""
	Content string "json:\"content\""
}

type Request struct {
	Model    string    "json:\"model\""
	Messages []Message "json:\"messages\""
}

type Choice struct {
	Message Message "json:\"message\""
}

type Response struct {
	Choices []Choice "json:\"choices\""
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody, _ := json.Marshal(Request{
		Model: c.model,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result Response
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response")
	}
	return result.Choices[0].Message.Content, nil
}
`
}

func (g *AgentGenerator) memoryContent() string {
	return `package memory

import (
	"context"
	"sync"
)

// Memory conversation memory
type Memory struct {
	mu       sync.RWMutex
	messages map[string][]Message
}

// Message conversation message
type Message struct {
	Role    string
	Content string
}

// NewMemory creates new memory
func NewMemory() *Memory {
	return &Memory{
		messages: make(map[string][]Message),
	}
}

// Add adds message to memory
func (m *Memory) Add(ctx context.Context, sessionID, role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[sessionID] = append(m.messages[sessionID], Message{Role: role, Content: content})
}

// Get gets conversation history
func (m *Memory) Get(ctx context.Context, sessionID string) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.messages[sessionID]
}
`
}

func (g *AgentGenerator) handlerContent() string {
	return `package handler

import (
	"encoding/json"
	"net/http"
	"context"
)

// Agent agent interface
type Agent interface {
	Run(ctx context.Context, input string) (string, error)
}

// Start starts HTTP server
func Start(a Agent) {
	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input string json:"input"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		response, err := a.Run(r.Context(), req.Input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		json.NewEncoder(w).Encode(map[string]string{"response": response})
	})
	
	http.ListenAndServe(":8080", nil)
}
`
}

func (g *AgentGenerator) promptContent() string {
	return `You are a helpful AI assistant.
Be concise and helpful in your responses.
`
}

func (g *AgentGenerator) configContent() string {
	return `server:
  address: ":8080"

llm:
  provider: "dmxapi"
  model: "qwen3.5-plus-free"
  api_key: "${DMXAPI_KEY}"
  base_url: "https://www.dmxapi.cn"

memory:
  max_messages: 100
`
}

func (g *AgentGenerator) goModContent() string {
	return fmt.Sprintf(`module %s

go %s
require (
	github.com/cloudwego/eino v0.1.0
)
`, g.projectName, GetGoVersion())
}

func (g *AgentGenerator) readmeContent() string {
	return fmt.Sprintf(`# %s - AI Agent

基于 CloudWeGo Eino 框架的 AI Agent 项目

## 运行

export OPENAI_API_KEY=your-key
go mod tidy
go run main.go

## API

POST /chat
{
  "input": "Hello"
}

Response:
{
  "response": "Hello! How can I help you?"
}
`, g.projectName)
}
