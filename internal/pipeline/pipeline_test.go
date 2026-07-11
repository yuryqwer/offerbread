package pipeline

import (
	"context"
	"testing"
)

func TestFanOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	src := make(chan int)
	dst1 := make(chan int)
	dst2 := make(chan int)

	go fanOut(ctx, src, dst1, dst2)

	go func() {
		for i := range 5 {
			src <- i
		}
		close(src)
	}()

	for i := range 5 {
		select {
		case val := <-dst1:
			select {
			case val2 := <-dst2:
				if val != val2 || val != i {
					t.Errorf("Expected same value from dst1 and dst2, got %d and %d", val, val2)
				}
			case <-ctx.Done():
				t.Fatal("Context cancelled while waiting for dst2")
			}
		case <-ctx.Done():
			t.Fatal("Context cancelled while waiting for dst1")
		}
	}
}
