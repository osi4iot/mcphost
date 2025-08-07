package mcphost

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type MCPHost interface {
	Run() error

	// Listar herramientas disponibles
	ListTools() ([]string, error)

    ListServers() ([]string, error)

	// Obtener información del host
	GetInfo() (*HostInfo, error)

	GiveMessages() []*schema.Message

	// Cerrar el host y limpiar recursos
	Close()

	// Obtener el ID único de esta instancia
	ID() string
}

type HostInfo struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	MCPServers []string  `json:"mcp_servers"`
	ToolNames  []string  `json:"tool_names,omitempty"`
	Model      string    `json:"model"`
	LastUsed   time.Time `json:"last_used"`
	Tokens     int       `json:"tokens"`
}

type mcpHost struct {
	id          string
	natsClient  *natsClient
	config      *HostConfig
	initialized bool
	serverNames []string
	toolNames   []string
	tokens      int
	lastUsed    time.Time
	ctx         context.Context
	cancel      context.CancelFunc
	messages    *[]*schema.Message
	mu          sync.RWMutex
}

// NewMCPHost crea una nueva instancia de MCPHost
func NewMCPHost(hostConfig *HostConfig) (MCPHost, error) {
	id := uuid.New().String()

	client, err := newNATSClient(hostConfig.NATSConfig, id)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var messages []*schema.Message
	if hostConfig.SavedMessages != nil {
		messages = hostConfig.SavedMessages
	} else {
		messages = []*schema.Message{}
	}

	return &mcpHost{
		id:          id,
		natsClient:  client,
		config:      hostConfig,
		ctx:         ctx,
		cancel:      cancel,
		initialized: false,
		tokens:      0,
		messages:    &messages,
	}, nil
}

func (h *mcpHost) Run() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.initialized {
		return fmt.Errorf("host already initialized")
	}

	h.initialized = true
	h.tokens = 0
	h.lastUsed = time.Now()

	h.RunMCPHost()

	return nil
}

func (h *mcpHost) ListTools() ([]string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.initialized {
		return nil, fmt.Errorf("host not initialized")
	}

	return h.toolNames, nil
}

func (h *mcpHost) ListServers() ([]string, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    if !h.initialized {
        return nil, fmt.Errorf("host not initialized")
    }

    return h.serverNames, nil
}

func (h *mcpHost) GetInfo() (*HostInfo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	servers := make([]string, 0, len(h.config.MCPServers))
	for name := range h.config.MCPServers {
		servers = append(servers, name)
	}

	status := "uninitialized"
	if h.initialized {
		status = "ready"
	}

	return &HostInfo{
		ID:         h.id,
		Status:     status,
		MCPServers: servers,
		Model:      h.config.Model,
		LastUsed:   h.lastUsed,
		Tokens:     h.tokens,
	}, nil
}

func (h *mcpHost) GiveMessages() []*schema.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return *h.messages
}

func (h *mcpHost) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.natsClient.Close()

	if h.cancel != nil {
		h.cancel()
	}

	h.initialized = false
}

func (h *mcpHost) ID() string {
	return h.id
}
