package rabbitmq

import (
	"log"
	"sync"

	"github.com/alhamdibahri/my-echo-app/internal/db"
	"github.com/alhamdibahri/my-echo-app/internal/worker"
	"github.com/streadway/amqp"
)

type Manager struct {
	conn      *amqp.Connection
	consumers map[string]*Consumer
	db        *db.Postgres
	mu        sync.Mutex
}

func NewManager(url string, db *db.Postgres) (*Manager, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &Manager{
		conn:      conn,
		consumers: make(map[string]*Consumer),
		db:        db,
	}, nil
}

func (m *Manager) CreateTenantQueue(tenantID string, concurrency int) error {
	ch, err := m.conn.Channel()
	if err != nil {
		return err
	}

	queueName := "tenant_" + tenantID + "_queue"

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	consumer := &Consumer{
		TenantID: tenantID,
		Queue:    queueName,
		StopCh:   make(chan struct{}),
		Channel:  ch,
		Pool:     worker.NewPool(concurrency, m.db),
	}

	consumer.StartConsuming()

	m.mu.Lock()
	m.consumers[tenantID] = consumer
	m.mu.Unlock()

	log.Printf("[Manager] Created consumer for tenant %s", tenantID)
	return nil
}

func (m *Manager) StopTenant(tenantID string) error {
	m.mu.Lock()
	c, ok := m.consumers[tenantID]
	m.mu.Unlock()
	if !ok {
		log.Printf("[Manager] not found tenant %s", tenantID)
		return nil
	}

	close(c.StopCh)
	c.Pool.Stop()

	if _, err := c.Channel.QueueDelete(c.Queue, false, false, false); err != nil {
		log.Printf("[Manager] QueueDelete error: %v", err)
	} else {
		log.Printf("[Manager] Queue deleted: %s", c.Queue)
	}

	c.Channel.Close()

	m.mu.Lock()
	delete(m.consumers, tenantID)
	m.mu.Unlock()

	log.Printf("[Manager] Stopped consumer for tenant %s", tenantID)
	return nil
}

func (m *Manager) UpdateConcurrency(tenantID string, workers int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.consumers[tenantID]
	if !ok {
		return nil
	}

	c.Pool.UpdateWorkers(workers)
	log.Printf("[Manager] Updated concurrency for tenant %s to %d", tenantID, workers)
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range m.consumers {
		close(c.StopCh)
		c.Pool.Stop()
		c.Channel.Close()
	}
	m.conn.Close()
}
