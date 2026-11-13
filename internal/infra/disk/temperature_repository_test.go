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

func temperatureRepositoryWithStoredPrograms(t *testing.T) disk.TemperatureProgramRepository {
	t.Helper()
	repo := disk.NewTemperatureProgramRepository(tempFile(t, "temperature_programs.yml"), tracer())
	hour1, err := program.ParseHour("20:12")
	require.NoError(t, err)
	hour2, err := program.ParseHour("21:12")
	require.NoError(t, err)
	hour3, err := program.ParseHour("22:12")
	require.NoError(t, err)
	seconds1, err := program.ParseSeconds(15)
	require.NoError(t, err)
	executions := []program.Execution{
		fixtures.ExecutionBuilder{Zones: []string{"1"}, Seconds: &seconds1}.Build(),
		fixtures.ExecutionBuilder{Zones: []string{"2"}}.Build(),
	}
	temperatures := []program.Temperature{
		fixtures.TemperatureBuilder{Temperature: new(float32(20.3)), Programs: []program.Program{
			fixtures.ProgramBuilder{Hour: &hour1}.Build(),
			fixtures.ProgramBuilder{Executions: executions}.Build(),
		}}.Build(),
		fixtures.TemperatureBuilder{Temperature: new(float32(19.3)), Programs: []program.Program{
			fixtures.ProgramBuilder{Hour: &hour2}.Build(),
		}}.Build(),
		fixtures.TemperatureBuilder{Temperature: new(float32(22.3)), Programs: []program.Program{
			fixtures.ProgramBuilder{Hour: &hour3, Executions: []program.Execution{
				fixtures.ExecutionBuilder{}.Build(),
				fixtures.ExecutionBuilder{}.Build(),
				fixtures.ExecutionBuilder{}.Build(),
			}}.Build(),
		}}.Build(),
	}
	for _, temp := range temperatures {
		require.NoError(t, repo.Save(t.Context(), &temp))
	}
	return repo
}

func TestTemperatureProgramRepositoryFindAll(t *testing.T) {
	t.Run(`Given a TemperatureProgramRepository with three saved temperatures, when FindAll method is called, then it returns the three temperatures`, func(t *testing.T) {
		t.Parallel()
		found, err := temperatureRepositoryWithStoredPrograms(t).FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, found, 3)
	})
}

func TestTemperatureProgramRepositoryFindByTemperatureAndHour(t *testing.T) {
	tests := []struct {
		name           string
		temperature    float32
		hour           string
		expectsProgram bool
	}{
		{
			name:        "with a temperature below every saved one, then it returns a not found error",
			temperature: 10,
			hour:        "15:10",
		},
		{
			name:        "with an hour without programs, then it returns a not found error",
			temperature: 22.3,
			hour:        "08:00",
		},
		{
			name:           "with a temperature above a saved one and its hour, then it returns one program per execution",
			temperature:    21,
			hour:           "15:10",
			expectsProgram: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a TemperatureProgramRepository with saved temperatures, when FindByTemperatureAndHour method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			hour, err := program.ParseHour(tt.hour)
			require.NoError(t, err)
			temp, err := temperatureRepositoryWithStoredPrograms(t).FindByTemperatureAndHour(t.Context(), tt.temperature, hour)
			if !tt.expectsProgram {
				require.ErrorAs(t, err, &errors.NotFoundError{})
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.temperature, temp.Temperature())
			require.Len(t, temp.Programs(), 2)
			require.Len(t, temp.Programs()[0].Executions(), 1)
			require.Equal(t, []string{"1"}, temp.Programs()[0].Executions()[0].Zones())
			require.Equal(t, 15, temp.Programs()[0].Executions()[0].Seconds().Int())
			require.Len(t, temp.Programs()[1].Executions(), 1)
			require.Equal(t, []string{"2"}, temp.Programs()[1].Executions()[0].Zones())
			require.Equal(t, 20, temp.Programs()[1].Executions()[0].Seconds().Int())
		})
	}
}

func TestTemperatureProgramRepositoryFindByTemperature(t *testing.T) {
	tests := []struct {
		name               string
		temperature        float32
		expectsTemperature bool
	}{
		{
			name:        "with a temperature that is not saved, then it returns a not found error",
			temperature: 40,
		},
		{
			name:               "with a saved temperature, then it returns its programs",
			temperature:        20.3,
			expectsTemperature: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a TemperatureProgramRepository with saved temperatures, when FindByTemperature method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			found, err := temperatureRepositoryWithStoredPrograms(t).FindByTemperature(t.Context(), tt.temperature)
			if !tt.expectsTemperature {
				require.ErrorAs(t, err, &errors.NotFoundError{})
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.temperature, found.Temperature())
			require.Len(t, found.Programs(), 2)
		})
	}
}

func TestTemperatureProgramRepositoryRemove(t *testing.T) {
	t.Run(`Given a TemperatureProgramRepository with saved temperatures, when Remove method is called with one of them, then it is no longer found`, func(t *testing.T) {
		t.Parallel()
		repo := temperatureRepositoryWithStoredPrograms(t)
		temperature := float32(20.3)
		require.NoError(t, repo.Remove(t.Context(), temperature))
		_, err := repo.FindByTemperature(t.Context(), temperature)
		require.ErrorAs(t, err, &errors.NotFoundError{})
	})
}
