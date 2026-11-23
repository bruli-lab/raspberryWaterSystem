package http

import "github.com/bruli/raspberryWaterSystem/internal/domain/program"

func applyProgramStatus(prg *program.Program, st *ProgramStatus) error {
	if st == nil {
		return nil
	}
	parsed, err := program.ParseStatus(string(*st))
	if err != nil {
		return err
	}
	prg.SetStatus(parsed)
	return nil
}
