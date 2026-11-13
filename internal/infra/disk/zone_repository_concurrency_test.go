//go:build infra

package disk_test

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestZoneRepositoryConcurrentSaves(t *testing.T) {
	tests := []struct {
		name  string
		zones int
	}{
		{name: "with many zones at the same time, then every zone is kept", zones: 20},
	}
	for _, tt := range tests {
		t.Run(`Given a zone repository,
		when Save method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := disk.NewZoneRepository(filepath.Join(t.TempDir(), "zones.yml"), noop.NewTracerProvider().Tracer("test"))
			var wg sync.WaitGroup
			for range tt.zones {
				wg.Go(func() {
					require.NoError(t, repo.Save(t.Context(), fixtures.ZoneBuilder{}.Build()))
				})
			}
			wg.Wait()

			zones, err := repo.FindAll(t.Context())
			require.NoError(t, err)
			require.Len(t, zones, tt.zones)
		})
	}
}
