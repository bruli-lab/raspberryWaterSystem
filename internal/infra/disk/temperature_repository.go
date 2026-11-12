package disk

import (
	"context"
	"maps"
	"strconv"

	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
)

type temperatureMap = map[float32]programMap

type TemperatureProgramRepository struct {
	path   string
	tracer trace.Tracer
}

func (t TemperatureProgramRepository) FindByTemperature(ctx context.Context, temperature float32) (*program.Temperature, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := t.tracer.Start(ctx, "TemperatureProgramRepository.FindByTemperature")
		defer span.End()
		temp := make(temperatureMap)
		if err := readYamlFile(t.path, &temp); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		byTemp, ok := temp[temperature]
		if !ok {
			err := errors.NewNotFoundError(strconv.FormatFloat(float64(temperature), 'f', -1, 32))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		tempPrgm, skipped := buildTemperatureProgram(temperature, byTemp)
		recordSkippedEntries(span, skipped)
		span.SetStatus(codes.Ok, "temperature found")
		return tempPrgm, nil
	}
}

func buildTemperatureProgram(temperature float32, prgms programMap) (*program.Temperature, []error) {
	programs, skipped := buildPrograms(prgms)
	tempPrgm := program.HydrateTemperature(temperature, programs)
	return &tempPrgm, skipped
}

func (t TemperatureProgramRepository) Remove(ctx context.Context, temperature float32) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := t.tracer.Start(ctx, "TemperatureProgramRepository.Remove")
		defer span.End()
		temp := make(temperatureMap)
		if err := readYamlFile(t.path, &temp); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		delete(temp, temperature)
		if err := writeYamlFile(t.path, &temp); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "temperature removed")
		return nil
	}
}

func (t TemperatureProgramRepository) Save(ctx context.Context, programs *program.Temperature) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := t.tracer.Start(ctx, "TemperatureProgramRepository.Save")
		defer span.End()
		temp := make(temperatureMap)
		if err := readYamlFile(t.path, &temp); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		temp[programs.Temperature()] = buildProgramMap(programs.Programs())
		if err := writeYamlFile(t.path, temp); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "temperature saved")
		return nil
	}
}

func (t TemperatureProgramRepository) FindAll(ctx context.Context) ([]program.Temperature, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := t.tracer.Start(ctx, "TemperatureProgramRepository.FindAll")
		defer span.End()
		temperature := make(temperatureMap)
		if err := readYamlFile(t.path, &temperature); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		temperatures, skipped := buildTemperaturePrograms(temperature)
		recordSkippedEntries(span, skipped)
		span.SetStatus(codes.Ok, "temperatures found")
		return temperatures, nil
	}
}

func buildTemperaturePrograms(temperature temperatureMap) ([]program.Temperature, []error) {
	prgms := make([]program.Temperature, 0, len(temperature))
	var skipped []error
	for temp, t := range temperature {
		programs, skippedPrograms := buildPrograms(t)
		skipped = append(skipped, skippedPrograms...)
		prgms = append(prgms, program.HydrateTemperature(temp, programs))
	}
	return prgms, skipped
}

func (t TemperatureProgramRepository) FindByTemperatureAndHour(ctx context.Context, temperature float32, hour program.Hour) (program.Temperature, error) {
	select {
	case <-ctx.Done():
		return program.Temperature{}, ctx.Err()
	default:
		_, span := t.tracer.Start(ctx, "TemperatureProgramRepository.FindByTemperatureAndHour")
		defer span.End()
		temp := make(temperatureMap)
		if err := readYamlFile(t.path, &temp); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return program.Temperature{}, err
		}
		programs := make(programMap, 0)
		for tempKey, progs := range temp {
			if temperature >= tempKey {
				maps.Copy(programs, progs)
			}
		}
		byHour, ok := programs[hour.String()]
		if !ok {
			err := errors.NotFoundError{}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return program.Temperature{}, err
		}
		span.SetStatus(codes.Ok, "temperature and hour found")
		return buildProgramTemperature(temperature, hour, byHour), nil
	}
}

func buildProgramTemperature(temperature float32, hour program.Hour, prgms []executions) program.Temperature {
	programs := make([]program.Program, len(prgms))
	for i, pd := range prgms {
		programs[i] = program.HydrateProgramWithoutFertilizer(hour, buildExecutions(pd))
	}
	return program.HydrateTemperature(temperature, programs)
}

func buildExecutions(pd executions) []program.Execution {
	return []program.Execution{program.HydrateExecution(program.HydrateSeconds(pd.Seconds), pd.Zones)}
}

func NewTemperatureProgramRepository(path string, tracer trace.Tracer) TemperatureProgramRepository {
	return TemperatureProgramRepository{path: path, tracer: tracer}
}
