package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/stretchr/testify/require"
)

func TestRetryUntilDone(t *testing.T) {
	tests := []struct {
		name          string
		failures      int32
		cancelAfter   time.Duration
		expectedCalls int32
	}{
		{
			name:          "with a function that succeeds, then it runs it once",
			expectedCalls: 1,
		},
		{
			name:          "with a function that fails twice, then it retries it until it succeeds",
			failures:      2,
			expectedCalls: 3,
		},
		{
			name:          "with a function that always fails and the context is cancelled, then it stops retrying it",
			failures:      1 << 30,
			cancelAfter:   50 * time.Millisecond,
			expectedCalls: -1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a function that can fail,
		when RetryUntilDone function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.cancelAfter > 0 {
				time.AfterFunc(tt.cancelAfter, cancel)
			}
			var calls atomic.Int32
			failingRun := func(_ context.Context) error {
				if calls.Add(1) <= tt.failures {
					return errors.New("bind: cannot assign requested address")
				}
				return nil
			}
			done := make(chan struct{})

			go func() {
				worker.RetryUntilDone(ctx, slog.New(slog.DiscardHandler), "test", time.Millisecond, failingRun)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("RetryUntilDone did not return")
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
