package disk

import (
	"context"

	"github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
)

type (
	programMap       = map[string][]executions
	storedProgramMap = map[string]storedProgram
	executions       = struct {
		Seconds int      `yaml:"seconds"`
		Zones   []string `yaml:"zones"`
	}
)

type storedProgram struct {
	Fertilizer bool         `yaml:"fertilizer"`
	Executions []executions `yaml:"executions"`
}

func (f *storedProgram) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		var legacy []executions
		if err := value.Decode(&legacy); err != nil {
			return err
		}
		*f = storedProgram{Executions: legacy}
		return nil
	}
	type plainStoredProgram storedProgram
	var plain plainStoredProgram
	if err := value.Decode(&plain); err != nil {
		return err
	}
	*f = storedProgram(plain)
	return nil
}

type ProgramRepository struct {
	filePath string
	tracer   trace.Tracer
}

func (d ProgramRepository) Remove(ctx context.Context, hour *program.Hour) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.Remove")
		defer span.End()
		savedData := make(storedProgramMap)
		if err := readYamlFile(d.filePath, &savedData); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		delete(savedData, hour.String())
		if err := writeYamlFile(d.filePath, savedData); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "program removed")
		return nil

	}
}

func (d ProgramRepository) Save(ctx context.Context, prg *program.Program) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.Save")
		defer span.End()
		savedData := make(storedProgramMap)
		if err := readYamlFile(d.filePath, &savedData); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		execs := make([]executions, len(prg.Executions()))
		for i, p := range prg.Executions() {
			execs[i] = buildExecution(&p)
		}
		savedData[prg.Hour().String()] = storedProgram{
			Fertilizer: prg.IsFertilizer(),
			Executions: execs,
		}
		if err := writeYamlFile(d.filePath, savedData); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "program saved")
		return nil
	}
}

func buildExecution(exec *program.Execution) executions {
	return executions{
		Seconds: exec.Seconds().Int(),
		Zones:   exec.Zones(),
	}
}

func buildProgramMap(programs []program.Program) programMap {
	dailyPrgms := make(programMap)
	for _, pr := range programs {
		pgrData := make([]executions, len(pr.Executions()))
		for i, p := range pr.Executions() {
			pgrData[i] = executions{
				Seconds: p.Seconds().Int(),
				Zones:   p.Zones(),
			}
		}
		dailyPrgms[pr.Hour().String()] = pgrData
	}
	return dailyPrgms
}

func (d ProgramRepository) FindAll(ctx context.Context) ([]program.Program, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.FindAll")
		defer span.End()
		dailyPgrms := make(storedProgramMap)
		if err := readYamlFile(d.filePath, &dailyPgrms); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		programs, skipped := buildStoredPrograms(dailyPgrms)
		recordSkippedEntries(span, skipped)
		span.SetStatus(codes.Ok, "programs found")
		return programs, nil
	}
}

func buildStoredPrograms(pr storedProgramMap) ([]program.Program, []error) {
	dailies := make([]program.Program, 0, len(pr))
	var skipped []error
	for hour, pg := range pr {
		p, err := buildProgram(pg.Executions, hour, pg.Fertilizer)
		if err != nil {
			skipped = append(skipped, err)
			continue
		}
		dailies = append(dailies, *p)
	}
	return dailies, skipped
}

func buildPrograms(pr programMap) ([]program.Program, []error) {
	dailies := make([]program.Program, 0, len(pr))
	var skipped []error
	for hour, pg := range pr {
		p, err := buildProgramWithoutFertilizer(pg, hour)
		if err != nil {
			skipped = append(skipped, err)
			continue
		}
		dailies = append(dailies, *p)
	}
	return dailies, skipped
}

func (d ProgramRepository) FindByHour(ctx context.Context, hour *program.Hour) (*program.Program, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.FindByHour")
		defer span.End()
		dailyPgrms := make(storedProgramMap)
		if err := readYamlFile(d.filePath, &dailyPgrms); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		pgr, ok := dailyPgrms[hour.String()]
		if !ok {
			err := errors.NewNotFoundError(hour.String())
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		prog, err := buildProgram(pgr.Executions, hour.String(), pgr.Fertilizer)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "program found")
		return prog, nil
	}
}

func buildProgram(pgr []executions, hour string, fertilizer bool) (*program.Program, error) {
	ho, err := program.ParseHour(hour)
	if err != nil {
		return nil, InvalidStoredEntryError{key: hour, cause: err}
	}
	prog := program.HydrateProgram(ho, fertilizer, buildProgramExecutions(pgr))
	return &prog, nil
}

func buildProgramWithoutFertilizer(pgr []executions, hour string) (*program.Program, error) {
	ho, err := program.ParseHour(hour)
	if err != nil {
		return nil, InvalidStoredEntryError{key: hour, cause: err}
	}
	prog := program.HydrateProgramWithoutFertilizer(ho, buildProgramExecutions(pgr))
	return &prog, nil
}

func buildProgramExecutions(pgr []executions) []program.Execution {
	exec := make([]program.Execution, len(pgr))
	for i, p := range pgr {
		exec[i] = program.HydrateExecution(program.HydrateSeconds(p.Seconds), p.Zones)
	}
	return exec
}

func NewProgramRepository(filePath string, tracer trace.Tracer) ProgramRepository {
	return ProgramRepository{filePath: filePath, tracer: tracer}
}
