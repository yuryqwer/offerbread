package pipeline

import (
	"context"
	"sync"
)

type Pipeline interface {
	Start() error
	Stop()
}

// fanOut 将 src channel 中的值复制到所有的 dsts channel 中
func fanOut[T any](ctx context.Context, wg *sync.WaitGroup, src <-chan T, dsts ...chan<- T) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case val, ok := <-src:
			if !ok {
				return
			}
			for _, dst := range dsts {
				select {
				case dst <- val:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
