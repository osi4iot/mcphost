// pkg/mcphost/host.go
package mcphost

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MCPHost es la interface principal que exportas
type MCPHost interface {
    // Inicializar el host con la configuración MCP
    Initialize(ctx context.Context, config *Config) error
    
    // Procesar un prompt y retornar respuesta
    ProcessPrompt(ctx context.Context, opts *PromptOptions) (*PromptResponse, error)
    
    // Listar herramientas disponibles
    ListTools(ctx context.Context) ([]string, error)
    
    // Obtener información del host
    GetInfo(ctx context.Context) (*HostInfo, error)
    
    // Cerrar el host y limpiar recursos
    Close()
    
    // Obtener el ID único de esta instancia
    ID() string
}

type HostInfo struct {
    ID            string            `json:"id"`
    Status        string            `json:"status"`
    MCPServers    []string          `json:"mcp_servers"`
    Model         string            `json:"model"`
    LastUsed      time.Time         `json:"last_used"`
    RequestCount  int               `json:"request_count"`
}

// Implementación concreta
type mcpHost struct {
    id           string
    natsClient   *natsClient
    config       *Config
    initialized  bool
    mu           sync.RWMutex
    requestCount int
    lastUsed     time.Time
}

// NewMCPHost crea una nueva instancia de MCPHost
func NewMCPHost(natsConfig *NATSConfig) (MCPHost, error) {
    id := uuid.New().String()
    
    client, err := newNATSClient(natsConfig, id)
    if err != nil {
        return nil, fmt.Errorf("failed to create NATS client: %w", err)
    }

    return &mcpHost{
        id:         id,
        natsClient: client,
    }, nil
}

func (h *mcpHost) Initialize(ctx context.Context, config *Config) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.config = config
    
    // Enviar configuración al worker vía NATS
    if err := h.natsClient.initialize(ctx, config); err != nil {
        return fmt.Errorf("failed to initialize MCP host: %w", err)
    }

    h.initialized = true
    return nil
}

func (h *mcpHost) ProcessPrompt(ctx context.Context, opts *PromptOptions) (*PromptResponse, error) {
    h.mu.Lock()
    if !h.initialized {
        h.mu.Unlock()
        return nil, fmt.Errorf("host not initialized")
    }
    h.requestCount++
    h.lastUsed = time.Now()
    h.mu.Unlock()

    // Usar configuración del host si no se especifica en opciones
    if opts.Model == "" && h.config.Model != "" {
        opts.Model = h.config.Model
    }
    if opts.MaxTokens == 0 && h.config.MaxTokens > 0 {
        opts.MaxTokens = h.config.MaxTokens
    }
    if opts.Temperature == nil && h.config.Temperature != nil {
        opts.Temperature = h.config.Temperature
    }

    return h.natsClient.processPrompt(ctx, opts)
}

func (h *mcpHost) ListTools(ctx context.Context) ([]string, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    if !h.initialized {
        return nil, fmt.Errorf("host not initialized")
    }

    //return h.natsClient.listTools(ctx)
	return []string{}, nil // Placeholder, implement actual tool listing
}

func (h *mcpHost) GetInfo(ctx context.Context) (*HostInfo, error) {
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
        ID:           h.id,
        Status:       status,
        MCPServers:   servers,
        Model:        h.config.Model,
        LastUsed:     h.lastUsed,
        RequestCount: h.requestCount,
    }, nil
}

func (h *mcpHost) Close() {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.natsClient.close()
}

func (h *mcpHost) ID() string {
    return h.id
}
