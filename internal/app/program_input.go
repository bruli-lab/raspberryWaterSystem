package app

import "github.com/bruli/raspberryWaterSystem/internal/domain/program"

type ProgramExecutionInput struct {
	Seconds int
	Zones   []string
}

type ProgramInput struct {
	Hour, Status string
	Executions   []ProgramExecutionInput
}

func parseProgramStatus(st string) (program.Status, error) {
	if st == "" {
		return program.EnabledStatus, nil
	}
	return program.ParseStatus(st)
}

func buildProgramExecutions(inputs []ProgramExecutionInput) ([]program.Execution, error) {
	executions := make([]program.Execution, len(inputs))
	for i, in := range inputs {
		sec, err := program.ParseSeconds(in.Seconds)
		if err != nil {
			return nil, err
		}
		exec, err := program.NewExecution(sec, in.Zones)
		if err != nil {
			return nil, err
		}
		executions[i] = exec
	}
	return executions, nil
}

func buildProgramsWithoutFertilizer(inputs []ProgramInput) ([]program.Program, error) {
	programs := make([]program.Program, len(inputs))
	for i, in := range inputs {
		hour, err := program.ParseHour(in.Hour)
		if err != nil {
			return nil, err
		}
		executions, err := buildProgramExecutions(in.Executions)
		if err != nil {
			return nil, err
		}
		st, err := parseProgramStatus(in.Status)
		if err != nil {
			return nil, err
		}
		prg, err := program.NewWithoutFertilizer(hour, executions)
		if err != nil {
			return nil, err
		}
		prg.SetStatus(st)
		programs[i] = *prg
	}
	return programs, nil
}
