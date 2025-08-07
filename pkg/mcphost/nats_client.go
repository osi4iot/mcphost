// pkg/mcphost/nats_client.go
package mcphost

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type natsClient struct {
    conn         *nats.Conn
    config       *NATSConfig
    instanceID   string
    subjectIn    string
    subjectOut   string
    responseMap  map[string]chan *natsResponse  // Para correlacionar responses
    mu           sync.RWMutex
}

type natsRequest struct {
    ID      string      `json:"id"`
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

type natsResponse struct {
    ID      string      `json:"id"`
    Type    string      `json:"type"`
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func newNATSClient(config *NATSConfig, instanceID string) (*natsClient, error) {
    conn, err := nats.Connect(config.URL, 
        nats.MaxReconnects(config.MaxReconnects),
        nats.ReconnectWait(config.ReconnectWait),
    )
    if err != nil {
        return nil, err
    }

    subjectIn := fmt.Sprintf("%s.%s.in", config.SubjectPrefix, instanceID)
    subjectOut := fmt.Sprintf("%s.%s.out", config.SubjectPrefix, instanceID)

    client := &natsClient{
        conn:        conn,
        config:      config,
        instanceID:  instanceID,
        subjectIn:   subjectIn,
        subjectOut:  subjectOut,
        responseMap: make(map[string]chan *natsResponse),
    }

    // Suscribirse a responses
    _, err = conn.Subscribe(subjectOut, client.handleResponse)
    if err != nil {
        conn.Close()
        return nil, err
    }

    return client, nil
}

func (c *natsClient) initialize(ctx context.Context, config *Config) error {
    request := &natsRequest{
        ID:      c.generateRequestID(),
        Type:    "initialize",
        Payload: config,
    }

    response, err := c.sendRequest(ctx, request)
    if err != nil {
        return err
    }

    if !response.Success {
        return fmt.Errorf("initialization failed: %s", response.Error)
    }

    return nil
}

func (c *natsClient) processPrompt(ctx context.Context, opts *PromptOptions) (*PromptResponse, error) {
    request := &natsRequest{
        ID:      c.generateRequestID(),
        Type:    "prompt",
        Payload: opts,
    }

    response, err := c.sendRequest(ctx, request)
    if err != nil {
        return nil, err
    }

    if !response.Success {
        return nil, fmt.Errorf("prompt processing failed: %s", response.Error)
    }

    // Convertir response.Data a PromptResponse
    var promptResponse PromptResponse
    data, _ := json.Marshal(response.Data)
    if err := json.Unmarshal(data, &promptResponse); err != nil {
        return nil, fmt.Errorf("failed to parse prompt response: %w", err)
    }

    return &promptResponse, nil
}

func (c *natsClient) sendRequest(ctx context.Context, request *natsRequest) (*natsResponse, error) {
    // Crear canal para la respuesta
    responseChan := make(chan *natsResponse, 1)
    
    c.mu.Lock()
    c.responseMap[request.ID] = responseChan
    c.mu.Unlock()

    // Cleanup
    defer func() {
        c.mu.Lock()
        delete(c.responseMap, request.ID)
        c.mu.Unlock()
        close(responseChan)
    }()

    // Serializar y enviar request
    data, err := json.Marshal(request)
    if err != nil {
        return nil, err
    }

    if err := c.conn.Publish(c.subjectIn, data); err != nil {
        return nil, err
    }

    // Esperar respuesta con timeout
    select {
    case response := <-responseChan:
        return response, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    case <-time.After(c.config.Timeout):
        return nil, fmt.Errorf("request timeout")
    }
}

func (c *natsClient) handleResponse(msg *nats.Msg) {
    var response natsResponse
    if err := json.Unmarshal(msg.Data, &response); err != nil {
        return
    }

    c.mu.RLock()
    responseChan, exists := c.responseMap[response.ID]
    c.mu.RUnlock()

    if exists && responseChan != nil {
        select {
        case responseChan <- &response:
        default:
            // Channel full or closed
        }
    }
}

func (c *natsClient) generateRequestID() string {
    return fmt.Sprintf("%s-%d", c.instanceID, time.Now().UnixNano())
}

func (c *natsClient) close()  {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Cancel all outstanding requests
    for _, responseChan := range c.responseMap {
        close(responseChan)
    }

    c.conn.Close()
}