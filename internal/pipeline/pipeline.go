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
func fanOut[T any](ctx context.Context, src <-chan T, dsts ...chan<- T) {
	for {
		select {
		case <-ctx.Done():
			return
		case val, ok := <-src:
			if !ok {
				return
			}
			var wg sync.WaitGroup
			wg.Add(len(dsts))
			for _, dst := range dsts {
				go func(dst chan<- T) {
					defer wg.Done()
					select {
					case dst <- val:
					case <-ctx.Done():
						return
					}
				}(dst)
			}
			wg.Wait()
		}
	}
}
