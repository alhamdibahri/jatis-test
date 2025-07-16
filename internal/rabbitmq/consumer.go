package rabbitmq

import (
	"log"

	"github.com/alhamdibahri/my-echo-app/internal/worker"
	"github.com/streadway/amqp"
)

type Consumer struct {
	TenantID string
	Queue    string
	StopCh   chan struct{}
	Channel  *amqp.Channel
	Pool     *worker.Pool
}

func (c *Consumer) StartConsuming() {
	msgs, err := c.Channel.Consume(
		c.Queue, "", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("[Consumer %s] failed to consume: %v", c.TenantID, err)
	}

	c.Pool.Start()

	go func() {
		for {
			select {
			case msg := <-msgs:
				c.Pool.Submit(worker.Job{
					TenantID: c.TenantID,
					Message:  msg.Body,
				})
			case <-c.StopCh:
				log.Printf("[Consumer %s] stopping", c.TenantID)
				return
			}
		}
	}()
}
