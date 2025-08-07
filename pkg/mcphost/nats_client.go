package mcphost

import (
	"github.com/nats-io/nats.go"
)

type natsClient struct {
    conn         *nats.Conn
    config       *NATSConfig
    instanceID   string
    subjectIn    string
    subjectOut   string
}


func newNATSClient(config *NATSConfig, instanceID string) (*natsClient, error) {
	conn, err := nats.Connect(
		config.ServersURL,
		nats.UserInfo(config.Username, config.Password),
	)
    if err != nil {
        return nil, err
    }

    client := &natsClient{
        conn:        conn,
        config:      config,
        instanceID:  instanceID,
        subjectIn:   config.SubjectIn,
        subjectOut:  config.SubjectOut,
    }

    return client, nil
}

func (c *natsClient) Close() {
    if c.conn != nil {
        c.conn.Close()
    }
}