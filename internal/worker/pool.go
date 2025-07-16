package worker

import (
	"log"
	"sync"

	"github.com/alhamdibahri/my-echo-app/internal/db"
)

type Job struct {
	TenantID string
	Message  []byte
}

type Pool struct {
	Workers int
	jobs    chan Job
	wg      sync.WaitGroup
	stopCh  chan struct{}
	DB      *db.Postgres
}

func NewPool(workers int, db *db.Postgres) *Pool {
	return &Pool{
		Workers: workers,
		jobs:    make(chan Job, 100),
		stopCh:  make(chan struct{}),
		DB:      db,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.Workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case job := <-p.jobs:
			if err := p.DB.EnsurePartitionForTenant(job.TenantID); err != nil {
				log.Printf("[Worker] failed to ensure partition: %v", err)
				continue
			}
			if err := p.DB.InsertMessage(job.TenantID, job.Message); err != nil {
				log.Printf("[Worker] Failed insert: %v", err)
			} else {
				log.Printf("[Worker] Inserted message for tenant %s", job.TenantID)
			}
		case <-p.stopCh:
			return
		}
	}
}

func (p *Pool) Submit(job Job) {
	p.jobs <- job
}

func (p *Pool) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

func (p *Pool) UpdateWorkers(n int) {
	p.Stop()
	p.Workers = n
	p.stopCh = make(chan struct{})
	p.wg = sync.WaitGroup{}
	p.Start()
}
