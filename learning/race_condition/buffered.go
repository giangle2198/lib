package race_condition

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// In a relay race of runner
type bufferedChannel struct {
	tasks chan string
	wg    sync.WaitGroup
}

func NewBufferedChannel(numGR, tLoad int) {
	b := &bufferedChannel{
		tasks: make(chan string, tLoad),
	}

	b.wg.Add(numGR)
	for gr := 1; gr <= numGR; gr++ {
		go b.worker(b.tasks, gr)
	}

	for post := 1; post <= tLoad; post++ {
		b.tasks <- fmt.Sprintf("Task: %d", post)
	}

	close(b.tasks)

	b.wg.Wait()
}

func (u *bufferedChannel) worker(tasks chan string, worker int) {
	defer u.wg.Done()

	for {
		// Wait for work to be assigned
		task, ok := <-tasks
		if !ok {
			// This means the channel is empty and closed
			fmt.Printf("Worker: %d : Shutting Down\n", worker)
			return
		}

		// Display we are starting the work
		fmt.Printf("Worker: %d : Started %s\n", worker, task)

		sleep := rand.Int63n(100)
		time.Sleep(time.Duration(sleep) * time.Microsecond)

		fmt.Printf("Worker: %d : Complete %s\n", worker, task)
	}
}
