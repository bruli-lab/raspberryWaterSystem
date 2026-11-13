package memory_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/memory"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestStatusRepositoryFind(t *testing.T) {
	t.Run(`Given an empty status repository, when Find method is called, then it returns a not found error`, func(t *testing.T) {
		t.Parallel()
		_, err := memory.NewStatusRepository(tracer()).Find(t.Context())
		require.ErrorAs(t, err, &errors.NotFoundError{})
	})
}

func TestStatusRepositorySave(t *testing.T) {
	t.Run(`Given a status repository, when Save method is called, then Find returns the saved status`, func(t *testing.T) {
		t.Parallel()
		repo := memory.NewStatusRepository(tracer())
		st := fixtures.StatusBuilder{Active: true}.Build()
		require.NoError(t, repo.Save(t.Context(), st))
		found, err := repo.Find(t.Context())
		require.NoError(t, err)
		require.Equal(t, st, found)
		require.True(t, found.IsActive())
	})
}

func TestStatusRepositoryUpdate(t *testing.T) {
	t.Run(`Given a status repository with a saved status, when Update method is called, then Find returns the updated status`, func(t *testing.T) {
		t.Parallel()
		repo := memory.NewStatusRepository(tracer())
		st := fixtures.StatusBuilder{Active: true}.Build()
		require.NoError(t, repo.Save(t.Context(), st))
		updated := st
		updated.Update(fixtures.WeatherBuilder{Raining: true}.Build(), fixtures.LightBuilder{}.Build())
		updated.Deactivate()
		require.NoError(t, repo.Update(t.Context(), updated))
		found, err := repo.Find(t.Context())
		require.NoError(t, err)
		require.Equal(t, updated, found)
		require.False(t, found.IsActive())
	})
}

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
