package disk

import (
	"context"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type (
	zonesMap map[string]zoneData
	zoneData struct {
		Name   string `yaml:"name"`
		Relays []int  `yaml:"relays"`
	}
)

type ZoneRepository struct {
	filePath string
	tracer   trace.Tracer
}

func (z ZoneRepository) Update(ctx context.Context, zo *zone.Zone) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := z.tracer.Start(ctx, "ZoneRepository.Update")
		defer span.End()
		zones := make(zonesMap)
		if err := readYamlFile(z.filePath, &zones); err != nil {
			return err
		}
		zones[zo.Id()] = zoneData{
			Name:   zo.Name(),
			Relays: z.buildRelaysForYaml(zo.Relays()),
		}
		if err := writeYamlFile(z.filePath, zones); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "zone updated")
		return nil
	}
}

func (z ZoneRepository) FindAll(ctx context.Context) ([]*zone.Zone, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := z.tracer.Start(ctx, "ZoneRepository.FindAll")
		defer span.End()
		zones := make(zonesMap)
		if err := readYamlFile(z.filePath, &zones); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "zones found")
		return z.buildZones(zones), nil
	}
}

func (z ZoneRepository) Remove(ctx context.Context, zo *zone.Zone) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := z.tracer.Start(ctx, "ZoneRepository.Remove")
		defer span.End()
		zones := make(zonesMap)
		if err := readYamlFile(z.filePath, &zones); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		_, ok := zones[zo.Id()]
		if !ok {
			err := errors.NewNotFoundError(zo.Id())
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		delete(zones, zo.Id())
		if err := writeYamlFile(z.filePath, zones); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "zone removed")
		return nil
	}
}

func (z ZoneRepository) FindByID(ctx context.Context, id string) (*zone.Zone, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := z.tracer.Start(ctx, "ZoneRepository.FindByID")
		defer span.End()
		zones := make(zonesMap)
		if err := readYamlFile(z.filePath, &zones); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		zo, ok := zones[id]
		if !ok {
			err := errors.NewNotFoundError(id)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "zone found")
		return buildZone(id, zo), nil
	}
}

func buildZone(id string, zo zoneData) *zone.Zone {
	return zone.HydrateZone(id, zo.Name, buildRelays(zo.Relays))
}

func buildRelays(relays []int) []zone.Relay {
	rel := make([]zone.Relay, len(relays))
	for i, n := range relays {
		r, _ := zone.ParseRelay(n)
		rel[i] = r
	}
	return rel
}

func (z ZoneRepository) Save(ctx context.Context, zo *zone.Zone) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := z.tracer.Start(ctx, "ZoneRepository.Save")
		defer span.End()
		zones := make(zonesMap)
		if err := readYamlFile(z.filePath, &zones); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		zones[zo.Id()] = zoneData{
			Name:   zo.Name(),
			Relays: z.buildRelaysForYaml(zo.Relays()),
		}
		if err := writeYamlFile(z.filePath, zones); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "zone saved")
		return nil
	}
}

func (z ZoneRepository) buildRelaysForYaml(rel []zone.Relay) []int {
	relays := make([]int, len(rel))
	for i, re := range rel {
		relays[i] = re.Id().Int()
	}
	return relays
}

func (z ZoneRepository) buildZones(data zonesMap) []*zone.Zone {
	zones := make([]*zone.Zone, 0, len(data))
	for i, zo := range data {
		zones = append(zones, buildZone(i, zo))
	}
	return zones
}

func NewZoneRepository(filePath string, tracer trace.Tracer) ZoneRepository {
	return ZoneRepository{filePath: filePath, tracer: tracer}
}
