package grpc_client

import (
	loggerI "auth-service/domain/service/logger"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

type ClientFactory struct {
	clients map[string]*Client
	mutex   sync.RWMutex
	log     loggerI.Log
}

func NewClientFactory(log loggerI.Log, config ...*Config) *ClientFactory {
	cf := &ClientFactory{
		clients: make(map[string]*Client),
		log:     log,
	}
	if len(config) > 0 {
		for _, c := range config {
			cf.CreateClient(c)
		}
	}
	return cf
}

func (cf *ClientFactory) CreateClient(config *Config) (*Client, error) {
	cf.mutex.Lock()
	defer cf.mutex.Unlock()

	if client, exists := cf.clients[config.Name]; exists {
		if client.IsConnected() {
			return client, nil
		}
		client.Close()
	}

	client, err := NewClient(config, cf.log)
	if err != nil {
		return nil, fmt.Errorf("không thể tạo client '%s': %w", config.Name, err)
	}
	if client.IsConnected() {
		cf.log.Info(fmt.Sprintf("Kết nối thành công: %s", client.config.Name), zap.String("host", client.conn.Target()))
	} else {
		cf.log.Error(fmt.Sprintf("Kết nối thất bại: %s", client.conn.Target()))
	}
	cf.clients[config.Name] = client
	return client, nil
}

func (cf *ClientFactory) GetClient(name string) *Client {
	cf.mutex.RLock()
	defer cf.mutex.RUnlock()
	return cf.clients[name]
}

func (cf *ClientFactory) CloseAll() error {
	cf.mutex.Lock()
	defer cf.mutex.Unlock()

	var lastErr error
	for name, client := range cf.clients {
		if err := client.Close(); err != nil {
			lastErr = fmt.Errorf("không thể đóng client '%s': %w", name, err)
		}
		delete(cf.clients, name)
	}

	return lastErr
}

func (cf *ClientFactory) CloseClient(name string) error {
	cf.mutex.Lock()
	defer cf.mutex.Unlock()

	if client, exists := cf.clients[name]; exists {
		err := client.Close()
		delete(cf.clients, name)
		return err
	}

	return fmt.Errorf("không tìm thấy client '%s'", name)
}

func (cf *ClientFactory) ListClients() []string {
	cf.mutex.RLock()
	defer cf.mutex.RUnlock()

	var names []string
	for name := range cf.clients {
		names = append(names, name)
	}
	return names
}

func (cf *ClientFactory) HealthCheck() map[string]bool {
	cf.mutex.RLock()
	defer cf.mutex.RUnlock()

	health := make(map[string]bool)
	for name, client := range cf.clients {
		health[name] = client.IsConnected()
	}
	return health
}
