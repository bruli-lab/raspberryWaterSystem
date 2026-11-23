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
	programMap = map[string]storedProgram
	executions = struct {
		Seconds int      `yaml:"seconds"`
		Zones   []string `yaml:"zones"`
	}
)

type storedProgram struct {
	Fertilizer bool         `yaml:"fertilizer"`
	Status     string       `yaml:"status,omitempty"`
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
		defer lockFile(d.filePath)()
		savedData := make(programMap)
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
		defer lockFile(d.filePath)()
		savedData := make(programMap)
		if err := readYamlFile(d.filePath, &savedData); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		savedData[prg.Hour().String()] = buildStoredProgram(prg)
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

func buildStoredProgram(prg *program.Program) storedProgram {
	execs := make([]executions, len(prg.Executions()))
	for i, p := range prg.Executions() {
		execs[i] = buildExecution(&p)
	}
	return storedProgram{
		Fertilizer: prg.IsFertilizer(),
		Status:     prg.Status().String(),
		Executions: execs,
	}
}

func buildProgramMap(programs []program.Program) programMap {
	prgms := make(programMap, len(programs))
	for _, pr := range programs {
		prgms[pr.Hour().String()] = buildStoredProgram(&pr)
	}
	return prgms
}

func (d ProgramRepository) FindAll(ctx context.Context) ([]program.Program, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.FindAll")
		defer span.End()
		defer lockFile(d.filePath)()
		dailyPgrms := make(programMap)
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

func buildStoredPrograms(pr programMap) ([]program.Program, []error) {
	dailies := make([]program.Program, 0, len(pr))
	var skipped []error
	for hour, pg := range pr {
		p, err := buildProgram(pg, hour)
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
		defer lockFile(d.filePath)()
		dailyPgrms := make(programMap)
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
		prog, err := buildProgram(pgr, hour.String())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "program found")
		return prog, nil
	}
}

func buildProgram(pgr storedProgram, hour string) (*program.Program, error) {
	ho, st, err := parseStoredProgramKeys(pgr, hour)
	if err != nil {
		return nil, err
	}
	prog := program.HydrateProgram(ho, pgr.Fertilizer, buildProgramExecutions(pgr.Executions), st)
	return &prog, nil
}

func buildProgramWithoutFertilizer(pgr storedProgram, hour string) (*program.Program, error) {
	ho, st, err := parseStoredProgramKeys(pgr, hour)
	if err != nil {
		return nil, err
	}
	prog := program.HydrateProgramWithoutFertilizer(ho, buildProgramExecutions(pgr.Executions), st)
	return &prog, nil
}

func parseStoredProgramKeys(pgr storedProgram, hour string) (program.Hour, program.Status, error) {
	ho, err := program.ParseHour(hour)
	if err != nil {
		return program.Hour{}, "", InvalidStoredEntryError{key: hour, cause: err}
	}
	st, err := parseStoredStatus(pgr.Status)
	if err != nil {
		return program.Hour{}, "", InvalidStoredEntryError{key: hour, cause: err}
	}
	return ho, st, nil
}

func parseStoredStatus(st string) (program.Status, error) {
	if st == "" {
		return program.EnabledStatus, nil
	}
	return program.ParseStatus(st)
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
