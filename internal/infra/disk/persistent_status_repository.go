package disk

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sync"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type persistedStatus struct {
	Active bool `json:"active"`
}

type PersistentStatusRepository struct {
	inner      app.StatusRepository
	path       string
	tracer     trace.Tracer
	m          *sync.Mutex
	lastActive *bool
}

func (p PersistentStatusRepository) Find(ctx context.Context) (status.Status, error) {
	return p.inner.Find(ctx)
}

func (p PersistentStatusRepository) Save(ctx context.Context, st status.Status) error {
	ctx, span := p.tracer.Start(ctx, "PersistentStatusRepository.Save")
	defer span.End()
	p.m.Lock()
	defer p.m.Unlock()
	persisted, found, err := p.load()
	if err != nil {
		span.RecordError(err)
	}
	if found && !persisted.Active {
		st.Deactivate()
	}
	if err = p.inner.Save(ctx, st); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	active := st.IsActive()
	*p.lastActive = active
	span.SetStatus(codes.Ok, "status saved")
	return nil
}

func (p PersistentStatusRepository) Update(ctx context.Context, st status.Status) error {
	ctx, span := p.tracer.Start(ctx, "PersistentStatusRepository.Update")
	defer span.End()
	p.m.Lock()
	defer p.m.Unlock()
	if *p.lastActive != st.IsActive() {
		if err := writeJsonFile(p.path, persistedStatus{Active: st.IsActive()}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		*p.lastActive = st.IsActive()
	}
	if err := p.inner.Update(ctx, st); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "status updated")
	return nil
}

func (p PersistentStatusRepository) load() (persistedStatus, bool, error) {
	var persisted persistedStatus
	data, err := os.ReadFile(p.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return persisted, false, nil
		}
		return persisted, false, err
	}
	if len(data) == 0 {
		return persisted, false, nil
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		return persisted, false, err
	}
	return persisted, true, nil
}

func NewPersistentStatusRepository(inner app.StatusRepository, path string, tracer trace.Tracer) PersistentStatusRepository {
	active := true
	return PersistentStatusRepository{inner: inner, path: path, tracer: tracer, m: &sync.Mutex{}, lastActive: &active}
}
