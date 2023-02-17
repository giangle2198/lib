package concurrencypatterns

import (
	"errors"
	"io"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// pool manages a set of resources that can be shared safely by
// multiple goroutines. The resource being managed must implement
// the io.Closer interface
type pool struct {
	m         sync.Mutex
	resources chan io.Closer
	factory   func() (io.Closer, error)
	closed    bool
}

type Pool interface {
	// Acquire retrieves a resource from the pool
	Acquire() (io.Closer, error)
	// Release places a new resource onto the pool
	Release(io.Closer)
	// Close will shutdown the pool and close all existing resources
	Close()
}

var ErrPoolClosed = errors.New("pool has been closed")

func NewPool(fn func() (io.Closer, error), size uint) Pool {
	if size <= 0 {
		zap.S().Panic("size value too small")
		return nil
	}

	return &pool{
		factory:   fn,
		resources: make(chan io.Closer, size),
	}
}

func (p *pool) Acquire() (io.Closer, error) {
	select {
	case r, ok := <-p.resources:
		log.Println("Acquire:", "Shared Resource")
		if !ok {
			return nil, ErrPoolClosed
		}
		return r, nil
	default:
		log.Println("Acquire:", "New Resource")
		return p.factory()
	}
}

func (p *pool) Release(r io.Closer) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		r.Close()
		return
	}

	select {
	case p.resources <- r:
		log.Println("Release:", "In Queue")

	default:
		log.Println("Release:", "Closing")
		r.Close()
	}
}

func (p *pool) Close() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.resources)
	for r := range p.resources {
		r.Close()
	}
}

type dbConnection struct {
	ID int32
}

var idCounter int32

func (conn *dbConnection) Close() error {
	log.Println("Close: Connection", conn.ID)
	return nil
}

func createConnection() (io.Closer, error) {
	id := atomic.AddInt32(&idCounter, 1)
	log.Println("Create: New Connection", id)
	return &dbConnection{id}, nil
}

func ProcessPool(maxGoroutines int, pooledResources int) {

	var wg sync.WaitGroup
	wg.Add(maxGoroutines)

	p := NewPool(createConnection, uint(pooledResources))

	for query := 0; query < maxGoroutines; query++ {
		go func(q int) {
			performQueries(q, p)
			wg.Done()
		}(query)
	}

	wg.Wait()

	log.Println("Shutdown Program")
	p.Close()
}

func performQueries(query int, p Pool) {
	conn, err := p.Acquire()
	if err != nil {
		log.Println(err)
		return
	}

	defer p.Release(conn)

	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	log.Printf("QID[%d] CID[%d]\n", query, conn.(*dbConnection).ID)
}
