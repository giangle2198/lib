package race_condition

import (
	"fmt"
	"runtime"
	"sync"
)

type mutexT struct {
	counter int64
	wg      sync.WaitGroup
	mutex   sync.Mutex
}

func NewMutexFunc(numWG int64) {
	a := &mutexT{}
	a.wg.Add(int(numWG))

	for i := 0; i < int(numWG); i++ {
		go a.incCounter(i)
	}

	a.wg.Wait()

	fmt.Println("Final Mutex Counter:", a.counter)
}

func (a *mutexT) incCounter(id int) {
	defer a.wg.Done()

	for count := 0; count < 2; count++ {

		a.mutex.Lock()
		{
			value := a.counter

			runtime.Gosched()

			value++

			a.counter = value
		}
		a.mutex.Unlock()
		// Yield the thread and be placed back in queue
		runtime.Gosched()
	}
}
