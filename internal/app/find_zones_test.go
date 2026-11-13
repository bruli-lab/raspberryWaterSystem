package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestFindZonesHandle(t *testing.T) {
	zones := []*zone.Zone{fixtures.ZoneBuilder{}.Build(), fixtures.ZoneBuilder{}.Build()}
	tests := []struct {
		name           string
		zones          []*zone.Zone
		repoErr        error
		expectedResult any
	}{
		{
			name:    "and find all method returns an error, then it returns same error",
			repoErr: errors.New("disk failure"),
		},
		{
			name:           "and find all method returns zones, then it returns them",
			zones:          zones,
			expectedResult: zones,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FindZones query handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			zr := &ZoneRepositoryMock{
				FindAllFunc: func(_ context.Context) ([]*zone.Zone, error) {
					return tt.zones, tt.repoErr
				},
			}
			handler := app.NewFindZones(zr, tracer())

			result, err := handler.Handle(t.Context(), app.FindZonesQuery{})
			if tt.repoErr != nil {
				require.ErrorIs(t, err, tt.repoErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedResult, result)
		})
	}
}
