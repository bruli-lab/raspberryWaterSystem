package http

import "github.com/bruli/raspberryWaterSystem/internal/app"

func programExecutionInputs(executions []ExecutionRequest) []app.ProgramExecutionInput {
	inputs := make([]app.ProgramExecutionInput, len(executions))
	for i, ex := range executions {
		inputs[i] = app.ProgramExecutionInput{Seconds: ex.Seconds, Zones: ex.Zones}
	}
	return inputs
}

func programInputs(programs []UpdateProgramItem) []app.ProgramInput {
	inputs := make([]app.ProgramInput, len(programs))
	for i, p := range programs {
		inputs[i] = app.ProgramInput{
			Hour:       p.Hour,
			Status:     programStatusInput(p.Status),
			Executions: programExecutionInputs(p.Executions),
		}
	}
	return inputs
}

func programStatusInput(st *ProgramStatus) string {
	if st == nil {
		return ""
	}
	return string(*st)
}
