//go:build infra

package disk_test

import (
	"context"
	"os"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"

	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
)

func TestZoneRepository(t *testing.T) {
	t.Run(`Given a ZoneRepository, `, func(t *testing.T) {
		ctx := context.Background()
		path := "/tmp/zones.yml"
		defer populateFile(t, path)
		repo := disk.NewZoneRepository(path, tracer())
		var savedZone *zone.Zone
		_ = savedZone
		t.Run(`when Save method is called,
		then it save a zone`, func(t *testing.T) {
			zo := fixtures.ZoneBuilder{}.Build()
			err := repo.Save(ctx, zo)
			require.NoError(t, err)
			savedZone = zo
		})
		t.Run(`when FindByID method is called,`, func(t *testing.T) {
			t.Run(`with an invalid id,
			then it returns a not found error`, func(t *testing.T) {
				_, err := repo.FindByID(ctx, uuid.New().String())
				require.ErrorAs(t, err, &errors.NotFoundError{})
			})
			t.Run(`with a valid id,
			then it returns the zone`, func(t *testing.T) {
				zo, err := repo.FindByID(ctx, savedZone.Id())
				require.NoError(t, err)
				require.Equal(t, savedZone, zo)
			})
		})
		t.Run(`when Remove method is called`, func(t *testing.T) {
			t.Run(`with an invalid zone,
			then it returns a not found error`, func(t *testing.T) {
				zo := fixtures.ZoneBuilder{}.Build()
				err := repo.Remove(ctx, zo)
				require.ErrorAs(t, err, &errors.NotFoundError{})
			})
			t.Run(`with a valid zone,
			then it returns nil`, func(t *testing.T) {
				err := repo.Remove(ctx, savedZone)
				require.NoError(t, err)
				_, errFind := repo.FindByID(ctx, savedZone.Id())
				require.ErrorAs(t, errFind, &errors.NotFoundError{})
			})
		})
	})
}

func populateFile(t *testing.T, path string) {
	if _, err := os.Stat(path); err == nil {
		if !os.IsNotExist(err) {
			err := os.Remove(path)
			require.NoError(t, err)
		}
	}
}

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
