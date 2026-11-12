package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestForwardWithoutBlocking(t *testing.T) {
	tests := []struct {
		name  string
		items int
	}{
		{name: "with a single item, then it is forwarded", items: 1},
		{name: "with more items than any consumer buffer while the consumer is busy, then producers never block and order is kept", items: 1000},
	}
	for _, tt := range tests {
		t.Run(`Given an events queue without a reader,
		when items are sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			in := make(chan int)
			out := make(chan int)
			go forwardWithoutBlocking(ctx, in, out)

			sent := make(chan struct{})
			go func() {
				for i := range tt.items {
					in <- i
				}
				close(sent)
			}()
			select {
			case <-sent:
			case <-time.After(2 * time.Second):
				require.FailNow(t, "producer blocked while the consumer was busy")
			}

			for i := range tt.items {
				select {
				case got := <-out:
					require.Equal(t, i, got)
				case <-time.After(2 * time.Second):
					require.FailNow(t, "item not forwarded")
				}
			}
		})
	}
}
