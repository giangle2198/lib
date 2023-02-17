package concurrencypatterns

import (
	"log"
	"sync"
	"time"
)

type Worker interface {
	Task()
}

type workerPool struct {
	works chan Worker
	wg    sync.WaitGroup
}

func NewWorkerPool(maxGoroutines int) *workerPool {
	p := &workerPool{
		works: make(chan Worker),
	}

	p.wg.Add(maxGoroutines)
	for i := 0; i < maxGoroutines; i++ {
		go func() {
			for w := range p.works {
				w.Task()
			}
			p.wg.Done()
		}()
	}

	return p
}

func (p *workerPool) Run(w Worker) {
	p.works <- w
}

func (p *workerPool) Shutdown() {
	close(p.works)
	p.wg.Wait()
}

var names = []string{
	"steve",
	"bob",
	"mary",
	"therese",
	"jason",
}

type namePrinter struct {
	name string
}

func (m *namePrinter) Task() {
	log.Println(m.name)
	time.Sleep(time.Second)
}

func ProcessWorkerPool(maxGoroutines int) {
	p := NewWorkerPool(2)

	var wg sync.WaitGroup
	wg.Add(3 * len(names))

	for i := 0; i < 3; i++ {
		for _, name := range names {
			np := namePrinter{
				name: name,
			}

			go func() {
				p.Run(&np)
				wg.Done()
			}()
		}
	}

	wg.Wait()

	p.Shutdown()
}
