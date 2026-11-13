package solar_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/solar"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestFallbackLightRepositoryFind(t *testing.T) {
	errTest := errors.New("api unreachable")
	primaryLight := fixtures.LightBuilder{}.Build()
	fallbackLight := fixtures.LightBuilder{}.Build()
	tests := []struct {
		name                    string
		primaryErr, fallbackErr error
		expectedLight           *status.Light
		expectedErr             error
		expectedFallbackCalls   int
	}{
		{
			name:          "and primary repository returns a light, then it returns that light without calling the fallback",
			expectedLight: primaryLight,
		},
		{
			name:                  "and primary repository returns an error, then it returns the fallback light",
			primaryErr:            errTest,
			expectedLight:         fallbackLight,
			expectedFallbackCalls: 1,
		},
		{
			name:                  "and both repositories return an error, then it returns the fallback error",
			primaryErr:            errTest,
			fallbackErr:           solar.NoSunriseError{},
			expectedErr:           solar.NoSunriseError{},
			expectedFallbackCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FallbackLightRepository,
		when Find method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			primary := &LightRepositoryMock{
				FindFunc: func(_ context.Context, _ time.Time) (*status.Light, error) {
					if tt.primaryErr != nil {
						return nil, tt.primaryErr
					}
					return primaryLight, nil
				},
			}
			fallback := &LightRepositoryMock{
				FindFunc: func(_ context.Context, _ time.Time) (*status.Light, error) {
					if tt.fallbackErr != nil {
						return nil, tt.fallbackErr
					}
					return fallbackLight, nil
				},
			}
			repo := solar.NewFallbackLightRepository(primary, fallback, noop.NewTracerProvider().Tracer("test"), slog.New(slog.DiscardHandler))

			light, err := repo.Find(t.Context(), time.Now())
			require.Len(t, fallback.FindCalls(), tt.expectedFallbackCalls)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
			require.Same(t, tt.expectedLight, light)
		})
	}
}
