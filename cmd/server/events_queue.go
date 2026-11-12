package main

import "context"

func forwardWithoutBlocking[T any](ctx context.Context, in <-chan T, out chan<- T) {
	var pending []T
	for {
		var next T
		var sendCh chan<- T
		if len(pending) > 0 {
			next = pending[0]
			sendCh = out
		}
		select {
		case <-ctx.Done():
			return
		case item := <-in:
			pending = append(pending, item)
		case sendCh <- next:
			var zero T
			pending[0] = zero
			pending = pending[1:]
		}
	}
}
