//go:build infra

package disk_test

import (
	"path/filepath"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}

func tempFile(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

func zoneRepositoryWith(t *testing.T, zones ...*zone.Zone) disk.ZoneRepository {
	t.Helper()
	repo := disk.NewZoneRepository(tempFile(t, "zones.yml"), tracer())
	for _, zo := range zones {
		require.NoError(t, repo.Save(t.Context(), zo))
	}
	return repo
}

func TestZoneRepositoryFindByID(t *testing.T) {
	saved := fixtures.ZoneBuilder{}.Build()
	tests := []struct {
		name        string
		id          string
		expectsZone bool
	}{
		{
			name: "with an unknown id, then it returns a not found error",
			id:   uuid.NewString(),
		},
		{
			name:        "with the id of a saved zone, then it returns the zone",
			id:          saved.Id(),
			expectsZone: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ZoneRepository with a saved zone, when FindByID method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := zoneRepositoryWith(t, saved)
			found, err := repo.FindByID(t.Context(), tt.id)
			if !tt.expectsZone {
				require.ErrorAs(t, err, &errors.NotFoundError{})
				return
			}
			require.NoError(t, err)
			require.Equal(t, saved, found)
		})
	}
}

func TestZoneRepositoryFindAll(t *testing.T) {
	t.Run(`Given a ZoneRepository with two saved zones, when FindAll method is called, then it returns both zones`, func(t *testing.T) {
		t.Parallel()
		repo := zoneRepositoryWith(t, fixtures.ZoneBuilder{}.Build(), fixtures.ZoneBuilder{}.Build())
		zones, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, zones, 2)
	})
}

func TestZoneRepositoryRemove(t *testing.T) {
	saved := fixtures.ZoneBuilder{}.Build()
	tests := []struct {
		name          string
		zone          *zone.Zone
		expectsRemove bool
	}{
		{
			name: "with a zone that is not saved, then it returns a not found error",
			zone: fixtures.ZoneBuilder{}.Build(),
		},
		{
			name:          "with a saved zone, then it removes it",
			zone:          saved,
			expectsRemove: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ZoneRepository with a saved zone, when Remove method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := zoneRepositoryWith(t, saved)
			err := repo.Remove(t.Context(), tt.zone)
			if !tt.expectsRemove {
				require.ErrorAs(t, err, &errors.NotFoundError{})
				return
			}
			require.NoError(t, err)
			_, err = repo.FindByID(t.Context(), saved.Id())
			require.ErrorAs(t, err, &errors.NotFoundError{})
		})
	}
}
