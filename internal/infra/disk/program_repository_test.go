//go:build infra

package disk_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
)

func programRepositoryWith(t *testing.T, programs ...program.Program) disk.ProgramRepository {
	t.Helper()
	repo := disk.NewProgramRepository(tempFile(t, "programs.yml"), tracer())
	for _, prg := range programs {
		require.NoError(t, repo.Save(t.Context(), &prg))
	}
	return repo
}

func storedPrograms(t *testing.T) (program.Hour, []program.Program) {
	t.Helper()
	hour, err := program.ParseHour("12:15")
	require.NoError(t, err)
	fertilizerHour, err := program.ParseHour("10:30")
	require.NoError(t, err)
	executions := []program.Execution{
		fixtures.ExecutionBuilder{}.Build(),
		fixtures.ExecutionBuilder{}.Build(),
		fixtures.ExecutionBuilder{}.Build(),
	}
	return hour, []program.Program{
		fixtures.ProgramBuilder{Hour: &hour, Executions: executions}.Build(),
		fixtures.ProgramBuilder{Hour: &fertilizerHour, Fertilizer: true}.Build(),
		fixtures.ProgramBuilder{}.Build(),
	}
}

func TestProgramRepositoryFindAll(t *testing.T) {
	t.Run(`Given a ProgramRepository with three saved programs, when FindAll method is called, then it returns the three programs`, func(t *testing.T) {
		t.Parallel()
		_, programs := storedPrograms(t)
		repo := programRepositoryWith(t, programs...)
		found, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.ElementsMatch(t, programs, found)
	})
}

func TestProgramRepositoryFindByHour(t *testing.T) {
	unknownHour, err := program.ParseHour("08:00")
	require.NoError(t, err)
	tests := []struct {
		name           string
		hour           func(saved program.Hour) program.Hour
		expectsProgram bool
	}{
		{
			name: "with an hour without program, then it returns a not found error",
			hour: func(program.Hour) program.Hour { return unknownHour },
		},
		{
			name:           "with the hour of a saved program, then it returns that program with its executions",
			hour:           func(saved program.Hour) program.Hour { return saved },
			expectsProgram: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ProgramRepository with saved programs, when FindByHour method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			savedHour, programs := storedPrograms(t)
			repo := programRepositoryWith(t, programs...)
			hour := tt.hour(savedHour)
			found, err := repo.FindByHour(t.Context(), &hour)
			if !tt.expectsProgram {
				require.ErrorAs(t, err, &errors.NotFoundError{})
				return
			}
			require.NoError(t, err)
			require.Equal(t, programs[0], *found)
		})
	}
}

func TestProgramRepositoryRemove(t *testing.T) {
	t.Run(`Given a ProgramRepository with saved programs, when Remove method is called with the hour of a saved program, then it is no longer found`, func(t *testing.T) {
		t.Parallel()
		hour, programs := storedPrograms(t)
		repo := programRepositoryWith(t, programs...)
		require.NoError(t, repo.Remove(t.Context(), &hour))
		_, err := repo.FindByHour(t.Context(), &hour)
		require.ErrorAs(t, err, &errors.NotFoundError{})
		remaining, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, remaining, 2)
	})
}
