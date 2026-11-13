package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/stretchr/testify/require"
)

func TestStatusWorkers(t *testing.T) {
	now := time.Date(2026, time.November, 13, 8, 0, 0, 0, time.UTC)
	weath := fixtures.WeatherBuilder{Raining: true}.Build()
	errTest := errors.New("test error")
	run := map[string]func(context.Context, cqs.QueryHandler, cqs.CommandHandler) error{
		"InitStatus": func(ctx context.Context, qh cqs.QueryHandler, ch cqs.CommandHandler) error {
			return worker.InitStatus(ctx, qh, ch, now)
		},
		"UpdateStatus": worker.UpdateStatus,
	}
	expectedCmd := map[string]cqs.Command{
		"InitStatus":   app.CreateStatusCmd{StartedAt: now, Weather: weath},
		"UpdateStatus": app.UpdateStatusCmd{Weather: weath},
	}
	tests := []struct {
		name                    string
		queryResult             any
		queryErr                error
		commandErr              error
		expectedErr             error
		expectsUnexpectedResult bool
		expectsCmd              bool
	}{
		{
			name:        "and find weather query returns an error, then it returns same error without touching the status",
			queryErr:    errTest,
			expectedErr: errTest,
		},
		{
			name:                    "and find weather query returns something that is not a weather, then it returns an unexpected result error, status untouched",
			queryResult:             "not a weather",
			expectsUnexpectedResult: true,
		},
		{
			name:        "and the status command returns an error, then it returns same error",
			queryResult: weath,
			commandErr:  errTest,
			expectedErr: errTest,
			expectsCmd:  true,
		},
		{
			name:        "and everything works, then it sends the status command with the current weather",
			queryResult: weath,
			expectsCmd:  true,
		},
	}
	for workerName, runWorker := range run {
		for _, tt := range tests {
			t.Run(`Given a `+workerName+` worker, when it runs `+tt.name, func(t *testing.T) {
				t.Parallel()
				qh := &cqs.QueryHandlerMock{
					HandleFunc: func(_ context.Context, _ cqs.Query) (any, error) {
						return tt.queryResult, tt.queryErr
					},
				}
				ch := &cqs.CommandHandlerMock{
					HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
						return nil, tt.commandErr
					},
				}
				err := runWorker(t.Context(), qh, ch)
				switch {
				case tt.expectedErr != nil:
					require.ErrorIs(t, err, tt.expectedErr)
				case tt.expectsUnexpectedResult:
					var unexpected worker.UnexpectedQueryResultError
					require.ErrorAs(t, err, &unexpected)
				default:
					require.NoError(t, err)
				}
				if !tt.expectsCmd {
					require.Empty(t, ch.HandleCalls())
					return
				}
				require.Len(t, ch.HandleCalls(), 1)
				require.Equal(t, expectedCmd[workerName], ch.HandleCalls()[0].Cmd)
			})
		}
	}
}
