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
// 每次从 src channel 中读取一个值后，会并发地将该值发送到所有的 dsts channel 中
// 直到所有的 dsts channel 都接收到该值才会继续读取 src channel 中的下一个值
// 如果 ctx 被取消，或者 src channel 被关闭，则 fanOut 会退出
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
