//go:build infra

package disk_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
)

func TestExecutionLogRepository(t *testing.T) {
	tests := []struct {
		name string
		logs []program.ExecutionLog
	}{
		{
			name: "without saved logs, then FindAll returns an empty slice",
		},
		{
			name: "with three saved logs, then FindAll returns the three logs",
			logs: []program.ExecutionLog{
				fixtures.ExecutionLogBuilder{ZoneName: new("zone 1")}.Build(),
				fixtures.ExecutionLogBuilder{ZoneName: new("zone 2")}.Build(),
				fixtures.ExecutionLogBuilder{ZoneName: new("zone 3")}.Build(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given an ExecutionLogRepository, when Save and FindAll methods are called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := disk.NewExecutionLogRepository(tempFile(t, "execution_logs.json"), tracer())
			if tt.logs != nil {
				require.NoError(t, repo.Save(t.Context(), tt.logs))
			}
			found, err := repo.FindAll(t.Context())
			require.NoError(t, err)
			require.Len(t, found, len(tt.logs))
			for i, lo := range tt.logs {
				require.Equal(t, lo.ZoneName(), found[i].ZoneName())
				require.Equal(t, lo.Seconds(), found[i].Seconds())
				require.True(t, lo.ExecutedAt().Equal(found[i].ExecutedAt()))
			}
		})
	}
}
