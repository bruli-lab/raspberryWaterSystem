//go:build infra

package disk_test

import (
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
)

func weeklyRepositoryWithStoredWeeklies(t *testing.T) disk.WeeklyRepository {
	t.Helper()
	repo := disk.NewWeeklyRepository(tempFile(t, "weekly_programs.yml"), tracer())
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
		require.NoError(t, repo.Save(t.Context(), &w))
	}
	return repo
}

func TestWeeklyRepositoryFindAll(t *testing.T) {
	t.Run(`Given a WeeklyRepository with three saved days, when FindAll method is called, then it returns the three days`, func(t *testing.T) {
		t.Parallel()
		weeklies, err := weeklyRepositoryWithStoredWeeklies(t).FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, weeklies, 3)
	})
}

func TestWeeklyRepositoryFindByDayAndHourWithStoredWeeklies(t *testing.T) {
	tests := []struct {
		name           string
		day            time.Weekday
		hour           string
		expectsProgram bool
	}{
		{
			name: "with a day without programs, then it returns a not found error",
			day:  time.Saturday,
			hour: "08:00",
		},
		{
			name: "with an hour without programs on that day, then it returns a not found error",
			day:  time.Friday,
			hour: "08:00",
		},
		{
			name:           "with a day and hour that have a program, then it returns that program",
			day:            time.Friday,
			hour:           "15:10",
			expectsProgram: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a WeeklyRepository with saved days, when FindByDayAndHour method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			day := program.WeekDay(tt.day)
			hour, err := program.ParseHour(tt.hour)
			require.NoError(t, err)
			weekly, err := weeklyRepositoryWithStoredWeeklies(t).FindByDayAndHour(t.Context(), &day, &hour)
			if !tt.expectsProgram {
				require.ErrorAs(t, err, &errors.NotFoundError{})
				return
			}
			require.NoError(t, err)
			require.Equal(t, day, weekly.WeekDay())
			require.Equal(t, hour, weekly.Programs()[0].Hour())
		})
	}
}

func TestWeeklyRepositoryFindByDay(t *testing.T) {
	tests := []struct {
		name          string
		day           time.Weekday
		expectsWeekly bool
	}{
		{
			name: "with a day without programs, then it returns a not found error",
			day:  time.Sunday,
		},
		{
			name:          "with a saved day, then it returns its programs",
			day:           time.Friday,
			expectsWeekly: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a WeeklyRepository with saved days, when FindByDay method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			day := program.WeekDay(tt.day)
			found, err := weeklyRepositoryWithStoredWeeklies(t).FindByDay(t.Context(), &day)
			if !tt.expectsWeekly {
				require.ErrorAs(t, err, &errors.NotFoundError{})
				return
			}
			require.NoError(t, err)
			require.Equal(t, day, found.WeekDay())
		})
	}
}

func TestWeeklyRepositoryRemove(t *testing.T) {
	t.Run(`Given a WeeklyRepository with three saved days, when Remove method is called with one of them, then the other two remain`, func(t *testing.T) {
		t.Parallel()
		repo := weeklyRepositoryWithStoredWeeklies(t)
		day := program.WeekDay(time.Friday)
		require.NoError(t, repo.Remove(t.Context(), &day))
		remaining, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, remaining, 2)
		_, err = repo.FindByDay(t.Context(), &day)
		require.ErrorAs(t, err, &errors.NotFoundError{})
	})
}
