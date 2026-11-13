//go:build infra

package disk_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
)

func TestWeeklyRepositoryFindByDayAndHour(t *testing.T) {
	t.Run(`Given a Weekly repository with two executions at the same hour, when FindByDayAndHour method is called, then it returns each execution once`, func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		repo := disk.NewWeeklyRepository(filepath.Join(t.TempDir(), "weekly.yml"), tracer())
		day := program.WeekDay(time.Monday)
		hour, err := program.ParseHour("07:30")
		require.NoError(t, err)
		executions := []program.Execution{
			fixtures.ExecutionBuilder{Zones: []string{"a"}}.Build(),
			fixtures.ExecutionBuilder{Zones: []string{"b"}}.Build(),
		}
		weekly := fixtures.WeeklyBuilder{
			WeekDay:  &day,
			Programs: []program.Program{fixtures.ProgramBuilder{Hour: &hour, Executions: executions}.Build()},
		}.Build()
		require.NoError(t, repo.Save(ctx, &weekly))

		found, err := repo.FindByDayAndHour(ctx, &day, &hour)
		require.NoError(t, err)

		var zones []string
		for _, prg := range found.Programs() {
			for _, exec := range prg.Executions() {
				zones = append(zones, exec.Zones()...)
			}
		}
		require.Equal(t, []string{"a", "b"}, zones)
	})
}
