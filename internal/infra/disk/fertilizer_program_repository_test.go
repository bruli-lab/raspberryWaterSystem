//go:build infra

package disk_test

import (
	"context"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
)

func TestFertilizerProgramRepository(t *testing.T) {
	t.Run(`Given a FertilizerProgramRepository, `, func(t *testing.T) {
		ctx := context.Background()
		path := "/tmp/programs.yml"
		defer populateFile(t, path)
		repo := disk.NewFertilizerProgramRepository(path, tracer())
		hour, err := program.ParseHour("12:15")
		require.NoError(t, err)
		t.Run(`when Save method is called,
		then it save programs`, func(t *testing.T) {
			hour2, err := program.ParseHour("10:30")
			require.NoError(t, err)
			exec := []program.Execution{
				fixtures.ExecutionBuilder{}.Build(),
				fixtures.ExecutionBuilder{}.Build(),
				fixtures.ExecutionBuilder{}.Build(),
			}
			programs := []program.FertilizerProgram{
				fixtures.FertilizerProgramBuilder{Hour: &hour, Executions: exec}.Build(),
				fixtures.FertilizerProgramBuilder{
					Hour:       &hour2,
					Executions: nil,
					Fertilizer: true,
				}.Build(),
				fixtures.FertilizerProgramBuilder{}.Build(),
			}
			for _, p := range programs {
				err = repo.Save(ctx, &p)
				require.NoError(t, err)

			}
		})
		t.Run(`when FindAll method is called,
		then it returns a programs slice`, func(t *testing.T) {
			dailies, err := repo.FindAll(ctx)
			require.NoError(t, err)
			require.Len(t, dailies, 3)
		})
		t.Run(`when FindByHour method is called `, func(t *testing.T) {
			t.Run(`with an nonexistent hour,
			then it returns a not found error`, func(t *testing.T) {
				hour, err := program.ParseHour("08:00")
				require.NoError(t, err)
				_, err = repo.FindByHour(ctx, &hour)
				require.ErrorAs(t, err, &errors.NotFoundError{})
			})
			t.Run(`with an existent hour,
			then it returns valid program`, func(t *testing.T) {
				prg, err := repo.FindByHour(ctx, &hour)
				require.NoError(t, err)
				require.Equal(t, hour, prg.Hour())
				require.Len(t, prg.Executions(), 3)
			})
		})
		t.Run(`when Remove method is called,
		then it returns no error`, func(t *testing.T) {
			err = repo.Remove(ctx, &hour)
			require.NoError(t, err)
			_, err = repo.FindByHour(ctx, &hour)
			require.ErrorAs(t, err, &errors.NotFoundError{})
		})
	})
}
