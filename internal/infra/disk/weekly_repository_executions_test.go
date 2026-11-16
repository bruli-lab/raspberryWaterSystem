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
	"go.opentelemetry.io/otel/trace/noop"
)

func TestWeeklyRepositoryFindByDayAndHourWithSeveralExecutions(t *testing.T) {
	t.Run(`Given a Weekly repository with a program of two executions in the same hour,
	when FindByDayAndHour method is called,
	then it returns a single program with each execution once`, func(t *testing.T) {
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "weekly_programs.yml")
		repo := disk.NewWeeklyRepository(path, noop.NewTracerProvider().Tracer("test"))
		day := program.WeekDay(time.Wednesday)
		hour, err := program.ParseHour("07:30")
		require.NoError(t, err)
		executions := []program.Execution{
			fixtures.ExecutionBuilder{Zones: []string{"bonsai big"}}.Build(),
			fixtures.ExecutionBuilder{Zones: []string{"bonsai small"}}.Build(),
		}
		programs := []program.Program{fixtures.ProgramBuilder{Hour: &hour, Executions: executions}.Build()}
		weekly := fixtures.WeeklyBuilder{WeekDay: &day, Programs: programs}.Build()
		require.NoError(t, repo.Save(ctx, &weekly))

		found, err := repo.FindByDayAndHour(ctx, &day, &hour)

		require.NoError(t, err)
		require.Len(t, found.Programs(), 1)
		require.Equal(t, hour, found.Programs()[0].Hour())
		require.Equal(t, executions, found.Programs()[0].Executions())
	})
}
