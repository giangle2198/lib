package race_condition

import (
	"fmt"
	"math/rand"
	"sync"
)

// In game of tennis
type unbufferedChannel struct {
	court chan int
	wg    sync.WaitGroup
}

func NewUnbufferedChannel() {
	u := &unbufferedChannel{
		court: make(chan int),
	}

	players := []string{
		"Samsung",
		"Iphone",
	}

	u.wg.Add(2)

	for i := 0; i < 2; i++ {
		go u.player(players[i])
	}

	u.court <- 1

	u.wg.Wait()
}

func (u *unbufferedChannel) player(name string) {
	defer u.wg.Done()

	for {
		ball, ok := <-u.court

		if !ok {
			fmt.Printf("Players %s Won\n", name)
			return
		}

		n := rand.Intn(100)
		if n%13 == 0 {
			fmt.Printf("Players %s Missed\n", name)
			close(u.court)
			return
		}

		fmt.Printf("Player %s Hit %d\n", name, ball)
		ball++

		u.court <- ball
	}
}
