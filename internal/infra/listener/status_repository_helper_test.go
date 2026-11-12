//go:build infra

package listener_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/memory"
	"github.com/stretchr/testify/require"
)

func statusRepository(t *testing.T, clean bool) *memory.StatusRepository {
	t.Helper()
	repo := memory.NewStatusRepository(tracer())
	require.NoError(t, repo.Save(t.Context(), fixtures.StatusBuilder{Active: true, Clean: clean}.Build()))
	return repo
}
