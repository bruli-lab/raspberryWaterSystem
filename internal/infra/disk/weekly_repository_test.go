//go:build infra

package disk_test

import (
	"context"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
)

func TestWeeklyRepository(t *testing.T) {
	t.Run(`Given a Weekly repository,`, func(t *testing.T) {
		ctx := context.Background()
		path := "/tmp/weekly_programs.yml"
		defer populateFile(t, path)
		repo := disk.NewWeeklyRepository(path, tracer())
		t.Run(`when Save method is called,
		then it save weekly programs`, func(t *testing.T) {
			monday := program.WeekDay(time.Monday)
			tuesday := program.WeekDay(time.Tuesday)
			hour, err := program.ParseHour("20:12")
			require.NoError(t, err)
			executions := []program.Execution{
				fixtures.ExecutionBuilder{}.Build(),
				fixtures.ExecutionBuilder{}.Build(),
				fixtures.ExecutionBuilder{}.Build(),
			}
			programs := []program.Program{
				fixtures.ProgramBuilder{Hour: &hour}.Build(),
				fixtures.ProgramBuilder{Executions: executions}.Build(),
			}
			weeklies := []program.Weekly{
				fixtures.WeeklyBuilder{WeekDay: &monday, Programs: programs}.Build(),
				fixtures.WeeklyBuilder{WeekDay: &tuesday, Programs: programs}.Build(),
				fixtures.WeeklyBuilder{}.Build(),
			}
			for _, w := range weeklies {
				err = repo.Save(ctx, &w)
				require.NoError(t, err)
			}
		})
		t.Run(`when FindAll method is called,
		then it returns a weekly programs slice`, func(t *testing.T) {
			weekly, err := repo.FindAll(ctx)
			require.NoError(t, err)
			require.Len(t, weekly, 3)
		})
		t.Run(`when FindByDayAndHour method is called `, func(t *testing.T) {
			t.Run(`with an invalid day,
			then it returns not found error`, func(t *testing.T) {
				day := program.WeekDay(time.Saturday)
				hour, err := program.ParseHour("08:00")
				require.NoError(t, err)
				_, err = repo.FindByDayAndHour(ctx, &day, &hour)
				require.ErrorAs(t, err, &errors.NotFoundError{})
			})
			t.Run(`with an invalid hour,
			then it returns not found error`, func(t *testing.T) {
				day := program.WeekDay(time.Friday)
				hour, err := program.ParseHour("08:00")
				require.NoError(t, err)
				_, err = repo.FindByDayAndHour(ctx, &day, &hour)
				require.ErrorAs(t, err, &errors.NotFoundError{})
			})
			t.Run(`with a valid day and hour,
			then it returns valid weekly program`, func(t *testing.T) {
				day := program.WeekDay(time.Friday)
				hour, err := program.ParseHour("15:10")
				require.NoError(t, err)
				weekly, err := repo.FindByDayAndHour(ctx, &day, &hour)
				require.NoError(t, err)
				require.Equal(t, day, weekly.WeekDay())
				require.Equal(t, hour, weekly.Programs()[0].Hour())
			})
		})
		t.Run(`when FindByDay method is called`, func(t *testing.T) {
			t.Run(`with an invalid day,
			then returns a not found error`, func(t *testing.T) {
				day := program.WeekDay(time.Sunday)
				_, err := repo.FindByDay(ctx, &day)
				require.ErrorAs(t, err, &errors.NotFoundError{})
			})
			t.Run(`with a valid day,
			then it returns a valid program`, func(t *testing.T) {
				day := program.WeekDay(time.Friday)
				found, err := repo.FindByDay(ctx, &day)
				require.NoError(t, err)
				require.Equal(t, day, found.WeekDay())
			})
		})
		t.Run(`when Remove method is called,
		then it remove the weekly program`, func(t *testing.T) {
			day := program.WeekDay(time.Friday)
			err := repo.Remove(ctx, &day)
			require.NoError(t, err)
			list, err := repo.FindAll(ctx)
			require.NoError(t, err)
			require.Len(t, list, 2)
		})
	})
}
