package listener_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
)

func TestExecutePinsOnExecuteZoneListen(t *testing.T) {
	errPins := errors.New("pins failed")
	errLog := errors.New("log failed")
	errPublish := errors.New("publish failed")
	tests := []struct {
		name                                string
		executeCHErr, logsCHErr, publishErr error
		expectedErrs                        []error
		expectedCommands                    []string
	}{
		{
			name:             "and execute pins command handler returns an error, then it returns same error and neither saves the log nor notifies",
			executeCHErr:     errPins,
			expectedErrs:     []error{errPins},
			expectedCommands: []string{"pins"},
		},
		{
			name:             "and save execution logs command handler returns an error, then it still notifies and returns the log error",
			logsCHErr:        errLog,
			expectedErrs:     []error{errLog},
			expectedCommands: []string{"pins", "log", "publish"},
		},
		{
			name:             "and publish message command handler returns an error, then it still saves the log and returns the publish error",
			publishErr:       errPublish,
			expectedErrs:     []error{errPublish},
			expectedCommands: []string{"pins", "log", "publish"},
		},
		{
			name:             "and both the log and the publish fail, then it returns both errors",
			logsCHErr:        errLog,
			publishErr:       errPublish,
			expectedErrs:     []error{errLog, errPublish},
			expectedCommands: []string{"pins", "log", "publish"},
		},
		{
			name:             "and all services works fine, then it executes the pins, saves the log and notifies",
			expectedCommands: []string{"pins", "log", "publish"},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ExecutePinsOnExecuteZone listener, when Listen method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			var commands []string
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, cmd cqs.Command) ([]event.Event, error) {
					switch cmd.(type) {
					case app.ExecutePinsCmd:
						commands = append(commands, "pins")
						return nil, tt.executeCHErr
					case app.SaveExecutionLogCmd:
						commands = append(commands, "log")
						return nil, tt.logsCHErr
					case app.PublishMessageCmd:
						commands = append(commands, "publish")
						return nil, tt.publishErr
					default:
						return nil, nil
					}
				},
			}
			err := listener.NewExecutePinsOnExecuteZone(ch, tracer()).Listen(t.Context(), zone.Executed{
				ZoneName: "zone test",
				Seconds:  10,
			})
			require.Equal(t, tt.expectedCommands, commands)
			if len(tt.expectedErrs) == 0 {
				require.NoError(t, err)
				return
			}
			for _, expected := range tt.expectedErrs {
				require.ErrorIs(t, err, expected)
			}
		})
	}
}

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
