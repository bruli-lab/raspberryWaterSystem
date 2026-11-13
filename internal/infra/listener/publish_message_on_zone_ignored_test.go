package listener_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPublishMessageOnZoneIgnored(t *testing.T) {
	tests := []struct {
		name        string
		chErr       error
		expectedErr error
	}{
		{
			name:        "and publish message command returns an error, then it returns same error",
			chErr:       errors.New("disk full"),
			expectedErr: errors.New("disk full"),
		},
		{
			name: "then it publishes a message with the zone name and the reason",
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PublishMessageOnZoneIgnored listener,
		when Listen method is called with an ignored zone event `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			list := listener.NewPublishMessageOnZoneIgnored(ch, tracer())

			err := list.Listen(t.Context(), zone.Ignored{
				BasicEvent: event.NewBasicEvent(zone.IgnoredEventName, uuid.New(), "bbf"),
				ZoneName:   "bonsai big",
				Reason:     "it is raining",
			})

			require.Len(t, ch.HandleCalls(), 1)
			require.Equal(t, app.PublishMessageCmd{Message: `Zone "bonsai big" ignored to send water. Reason: it is raining`}, ch.HandleCalls()[0].Cmd)
			if tt.expectedErr != nil {
				require.EqualError(t, err, tt.expectedErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
