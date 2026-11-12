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
		span.SetStatus(codes.Ok, "temperature found")
		return buildTemperatureProgram(temperature, byTemp), nil
	}
}

func buildTemperatureProgram(temperature float32, prgms programMap) *program.Temperature {
	var tempPrgm program.Temperature
	tempPrgm.Hydrate(temperature, buildPrograms(prgms))
	return &tempPrgm
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
		span.SetStatus(codes.Ok, "temperatures found")
		return buildTemperaturePrograms(temperature), nil
	}
}

func buildTemperaturePrograms(temperature temperatureMap) []program.Temperature {
	prgms := make([]program.Temperature, 0, len(temperature))
	for temp, t := range temperature {
		var prg program.Temperature
		prg.Hydrate(temp, buildPrograms(t))
		prgms = append(prgms, prg)
	}
	return prgms
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
	var temp program.Temperature
	var pg program.Program
	for i, pd := range prgms {
		pg.Hydrate(hour, buildExecutions(pd))
		programs[i] = pg
	}
	temp.Hydrate(temperature, programs)
	return temp
}

func buildExecutions(pd executions) []program.Execution {
	executions := make([]program.Execution, 1)
	var execution program.Execution
	sec := program.HydrateSeconds(pd.Seconds)
	execution.Hydrate(sec, pd.Zones)
	executions[0] = execution
	return executions
}

func NewTemperatureProgramRepository(path string, tracer trace.Tracer) TemperatureProgramRepository {
	return TemperatureProgramRepository{path: path, tracer: tracer}
}
