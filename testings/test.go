package main

import (
	"fmt"
)

func main() {
	chan1 := make(chan int)
	chan2 := make(chan int)

	go func() {
		for i := 0; i < 10; i++ {
			chan1 <- i
		}
		close(chan1)
	}()

	go func() {
		for i := 0; i < 10; i++ {
			chan2 <- i
		}
		close(chan2)
	}()

	c := combineChans(chan1, chan2)

	for combinedInt := range c {
		fmt.Printf("(%d, %d)\n", combinedInt.int1, combinedInt.int2)
	}
}

type CombinedInts struct {
	int1 int
	int2 int
}

func combineChans(chan1, chan2 <-chan int) chan CombinedInts {
	combined := make(chan CombinedInts)
	go func() {
		for {
			select {
			case int1, ok := <-chan1:
				if !ok {
					return
				}
				int2, ok := <-chan2
				if !ok {
					return
				}
				combined <- CombinedInts{int1, int2}

			case int2, ok := <-chan2:
				if !ok {
					return
				}
				int1, ok := <-chan1
				if !ok {
					return
				}
				combined <- CombinedInts{int1, int2}
			}
		}
		close(combined)
	}()
	return combined
}
