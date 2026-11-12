package main

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSuperviseWorker(t *testing.T) {
	tests := []struct {
		name          string
		panics        int32
		cancelAfter   time.Duration
		expectedCalls int32
	}{
		{
			name:          "with a worker that finishes, then it runs it once",
			expectedCalls: 1,
		},
		{
			name:          "with a worker that panics twice, then it recovers, restarts it and returns when it finishes",
			panics:        2,
			expectedCalls: 3,
		},
		{
			name:          "with a worker that always panics and the context is cancelled, then it stops restarting it",
			panics:        1 << 30,
			cancelAfter:   50 * time.Millisecond,
			expectedCalls: -1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a supervised worker,
		when superviseWorker function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.cancelAfter > 0 {
				time.AfterFunc(tt.cancelAfter, cancel)
			}
			var calls atomic.Int32
			worker := func(_ context.Context) {
				if calls.Add(1) <= tt.panics {
					panic("sensor is nil")
				}
			}
			done := make(chan struct{})

			go func() {
				superviseWorkerWithDelay(ctx, slog.New(slog.DiscardHandler), "test", time.Millisecond, worker)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("superviseWorker did not return")
			}
			if tt.expectedCalls > 0 {
				require.Equal(t, tt.expectedCalls, calls.Load())
				return
			}
			require.ErrorIs(t, ctx.Err(), context.Canceled)
			require.Greater(t, calls.Load(), int32(1))
		})
	}
}
