package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/stretchr/testify/require"
)

func TestExecuteZoneWithStatusHandle(t *testing.T) {
	errTest := errors.New("")
	cmd := app.ExecuteZoneWithStatusCmd{
		Seconds: 367,
		ZoneID:  "name",
	}
	st := fixtures.StatusBuilder{Active: true}.Build()
	rainingWeather := fixtures.WeatherBuilder{Raining: true}.Build()
	statusRaining := fixtures.StatusBuilder{Weather: &rainingWeather}.Build()
	statusDeactivated := fixtures.StatusBuilder{Active: false}.Build()
	tests := []struct {
		name    string
		command cqs.Command
		expectedErr, findErr,
		stErr error
		zone          *zone.Zone
		expectedEvent string
		st            status.Status
	}{
		{
			name:        "with invalid command, then it returns an invalid command error",
			command:     invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and find zone returns an error, then it returns same error",
			command:     cmd,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and status repository returns an error, then it returns same error",
			command:     cmd,
			zone:        fixtures.ZoneBuilder{}.Build(),
			stErr:       errTest,
			expectedErr: errTest,
		},
		{
			name:          "and status repository returns a clean status, then it returns an ignored event",
			command:       cmd,
			zone:          fixtures.ZoneBuilder{}.Build(),
			st:            fixtures.StatusBuilder{Clean: true}.Build(),
			expectedEvent: zone.IgnoredEventName,
		},
		{
			name: "with fertilizer and status repository returns a clean status, then it returns an ignored event",
			command: app.ExecuteZoneWithStatusCmd{
				Seconds:        36,
				ZoneID:         "name",
				WithFertilizer: true,
			},
			zone:          fixtures.ZoneBuilder{}.Build(),
			st:            fixtures.StatusBuilder{Clean: true}.Build(),
			expectedEvent: zone.IgnoredEventName,
		},
		{
			name:        "and execute returns an error, then it returns an execute zone with status error",
			zone:        fixtures.ZoneBuilder{}.Build(),
			st:          st,
			expectedErr: app.ExecuteZoneWithStatusError{},
			command:     cmd,
		},
		{
			name: "and execute nil, then it returns an executed event",
			zone: fixtures.ZoneBuilder{}.Build(),
			st:   st,
			command: app.ExecuteZoneWithStatusCmd{
				Seconds: 36,
				ZoneID:  "name",
			},
			expectedEvent: zone.ExecutedEventName,
		},
		{
			name: "with fertilizer and execute returns error, then it returns an execute zone with status error",
			zone: fixtures.ZoneBuilder{}.Build(),
			st:   st,
			command: app.ExecuteZoneWithStatusCmd{
				Seconds:        360,
				ZoneID:         "name",
				WithFertilizer: true,
			},
			expectedErr: app.ExecuteZoneWithStatusError{},
		},
		{
			name: "with fertilizer and execute returns , then it returns a fertilizer zone executed event",
			zone: fixtures.ZoneBuilder{}.Build(),
			st:   st,
			command: app.ExecuteZoneWithStatusCmd{
				Seconds:        36,
				ZoneID:         "name",
				WithFertilizer: true,
			},
			expectedEvent: zone.FertilizerZoneExecutedEventName,
		},
		{
			name: "and its raining, then it returns an ignored event",
			zone: fixtures.ZoneBuilder{}.Build(),
			st:   statusRaining,
			command: app.ExecuteZoneWithStatusCmd{
				Seconds: 36,
				ZoneID:  "name",
			},
			expectedEvent: zone.IgnoredEventName,
		},
		{
			name: "and system is deactivated, then it returns an ignored event",
			zone: fixtures.ZoneBuilder{}.Build(),
			st:   statusDeactivated,
			command: app.ExecuteZoneWithStatusCmd{
				Seconds: 36,
				ZoneID:  "name",
			},
			expectedEvent: zone.IgnoredEventName,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an executeZone command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			zr := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return tt.zone, tt.findErr
				},
			}
			sr := &StatusRepositoryMock{}
			sr.FindFunc = func(_ context.Context) (status.Status, error) {
				return tt.st, tt.stErr
			}
			handler := app.NewExecuteZoneWithStatus(zr, sr, tracer())
			events, err := handler.Handle(context.Background(), tt.command)
			if err != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.Equal(t, tt.expectedErr, err)
			require.Len(t, events, 1)
			ev := events[0]
			require.Equal(t, tt.expectedEvent, ev.EventName())
		})
	}
}
