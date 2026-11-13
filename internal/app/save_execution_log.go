package app

import (
	"context"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const SaveExecutionLogCmdName = "saveExecutionLog"

const maxExecutionLogs = 20

type SaveExecutionLogCmd struct {
	ZoneName   string
	Seconds    program.Seconds
	ExecutedAt time.Time
}

func (s SaveExecutionLogCmd) Name() string {
	return SaveExecutionLogCmdName
}

type SaveExecutionLog struct {
	elr    ExecutionLogRepository
	tracer trace.Tracer
}

func (s SaveExecutionLog) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := s.tracer.Start(ctx, "SaveExecutionLogCmd")
	defer span.End()
	co, ok := cmd.(SaveExecutionLogCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(SaveExecutionLogCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	logs, err := s.elr.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	readedLogs := logs
	if len(logs) >= maxExecutionLogs {
		readedLogs = logs[len(logs)-maxExecutionLogs:]
	}
	log, err := program.NewExecutionLog(co.Seconds, co.ZoneName, co.ExecutedAt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, SaveExecutionLogError{m: err.Error()}
	}
	readedLogs = append(readedLogs, log)
	if err = s.elr.Save(ctx, readedLogs); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "execution log saved")
	return nil, nil
}

func NewSaveExecutionLog(elr ExecutionLogRepository, tracer trace.Tracer) SaveExecutionLog {
	return SaveExecutionLog{elr: elr, tracer: tracer}
}

type SaveExecutionLogError struct {
	m string
}

func (s SaveExecutionLogError) Error() string {
	return "failed to save execution log: " + s.m
}
