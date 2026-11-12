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
	programMap    = map[string][]executions
	fertilizerMap = map[string]fertilizerProgram
	executions    = struct {
		Seconds int      `yaml:"seconds"`
		Zones   []string `yaml:"zones"`
	}
)

type fertilizerProgram struct {
	Fertilizer bool         `yaml:"fertilizer"`
	Executions []executions `yaml:"executions"`
}

func (f *fertilizerProgram) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		var legacy []executions
		if err := value.Decode(&legacy); err != nil {
			return err
		}
		*f = fertilizerProgram{Executions: legacy}
		return nil
	}
	type plainFertilizerProgram fertilizerProgram
	var plain plainFertilizerProgram
	if err := value.Decode(&plain); err != nil {
		return err
	}
	*f = fertilizerProgram(plain)
	return nil
}

type FertilizerProgramRepository struct {
	filePath string
	tracer   trace.Tracer
}

func (d FertilizerProgramRepository) Remove(ctx context.Context, hour *program.Hour) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.Remove")
		defer span.End()
		savedData := make(fertilizerMap)
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

func (d FertilizerProgramRepository) Save(ctx context.Context, prg *program.FertilizerProgram) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.Save")
		defer span.End()
		savedData := make(fertilizerMap)
		if err := readYamlFile(d.filePath, &savedData); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		execs := make([]executions, len(prg.Executions()))
		for i, p := range prg.Executions() {
			execs[i] = buildExecution(&p)
		}
		savedData[prg.Hour().String()] = fertilizerProgram{
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

func (d FertilizerProgramRepository) FindAll(ctx context.Context) ([]program.FertilizerProgram, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.FindAll")
		defer span.End()
		dailyPgrms := make(fertilizerMap)
		if err := readYamlFile(d.filePath, &dailyPgrms); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "programs found")
		return buildFertilizerPrograms(dailyPgrms), nil
	}
}

func buildFertilizerPrograms(pr fertilizerMap) []program.FertilizerProgram {
	dailies := make([]program.FertilizerProgram, 0, len(pr))
	for hour, pg := range pr {
		p := buildFertilizerProgram(pg.Executions, hour, pg.Fertilizer)
		dailies = append(dailies, *p)
	}
	return dailies
}

func buildPrograms(pr programMap) []program.Program {
	dailies := make([]program.Program, 0, len(pr))
	for hour, pg := range pr {
		p := buildProgram(pg, hour)
		dailies = append(dailies, *p)
	}
	return dailies
}

func (d FertilizerProgramRepository) FindByHour(ctx context.Context, hour *program.Hour) (*program.FertilizerProgram, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := d.tracer.Start(ctx, "ProgramRepository.FindByHour")
		defer span.End()
		dailyPgrms := make(fertilizerMap)
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
		span.SetStatus(codes.Ok, "program found")
		return buildFertilizerProgram(pgr.Executions, hour.String(), pgr.Fertilizer), nil
	}
}

func buildFertilizerProgram(pgr []executions, hour string, fertilizer bool) *program.FertilizerProgram {
	var prog program.FertilizerProgram
	ho, _ := program.ParseHour(hour)
	exec := make([]program.Execution, len(pgr))
	for i, p := range pgr {
		var pExec program.Execution
		sec, _ := program.ParseSeconds(p.Seconds)
		pExec.Hydrate(sec, p.Zones)
		exec[i] = pExec
	}
	prog.Hydrate(ho, fertilizer, exec)
	return &prog
}

func buildProgram(pgr []executions, hour string) *program.Program {
	var prog program.Program
	ho, _ := program.ParseHour(hour)
	exec := make([]program.Execution, len(pgr))
	for i, p := range pgr {
		var pExec program.Execution
		sec, _ := program.ParseSeconds(p.Seconds)
		pExec.Hydrate(sec, p.Zones)
		exec[i] = pExec
	}
	prog.Hydrate(ho, exec)
	return &prog
}

func NewFertilizerProgramRepository(filePath string, tracer trace.Tracer) FertilizerProgramRepository {
	return FertilizerProgramRepository{filePath: filePath, tracer: tracer}
}
