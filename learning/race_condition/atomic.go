package race_condition

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

type atomicTest struct {
	counter int64
	wg      sync.WaitGroup
}

func NewAtomicFunc(numWG int64) {
	a := &atomicTest{}
	a.wg.Add(int(numWG))

	for i := 0; i < int(numWG); i++ {
		go a.incCounter(i)
	}

	a.wg.Wait()

	fmt.Println("Final Atomic Counter:", a.counter)
}

func (a *atomicTest) incCounter(id int) {
	defer a.wg.Done()

	for count := 0; count < 2; count++ {
		atomic.AddInt64(&a.counter, 1)

		// Yield the thread and be placed back in queue
		runtime.Gosched()
	}
}
