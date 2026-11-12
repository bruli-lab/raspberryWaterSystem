package app

import (
	"context"
	"errors"
	"sort"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const FindExecutionLogsQueryName = "findExecutionLogs"

var ErrInvalidExecutionsLogLimit = errors.New("invalid executions log limit")

type FindExecutionLogsQuery struct {
	Limit int
}

func (f FindExecutionLogsQuery) Name() string {
	return FindExecutionLogsQueryName
}

type FindExecutionLogs struct {
	elr    ExecutionLogRepository
	tracer trace.Tracer
}

func (f FindExecutionLogs) Handle(ctx context.Context, query cqs.Query) (any, error) {
	ctx, span := f.tracer.Start(ctx, "FindExecutionLogs")
	defer span.End()
	q, ok := query.(FindExecutionLogsQuery)
	if !ok {
		err := cqs.NewInvalidQueryError(FindExecutionLogsQueryName, query.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if q.Limit > maxExecutionLogs {
		err := ErrInvalidExecutionsLogLimit
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	logs, err := f.elr.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if q.Limit >= len(logs) {
		span.SetStatus(codes.Ok, "all execution logs found")
		return f.orderLogs(logs), nil
	}
	filtered := logs[len(logs)-q.Limit:]
	span.SetStatus(codes.Ok, "execution logs found")
	return f.orderLogs(filtered), nil
}

func (f FindExecutionLogs) orderLogs(logs []program.ExecutionLog) []program.ExecutionLog {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].ExecutedAt().After(logs[j].ExecutedAt())
	})
	return logs
}

func NewFindExecutionLogs(elr ExecutionLogRepository, tracer trace.Tracer) FindExecutionLogs {
	return FindExecutionLogs{elr: elr, tracer: tracer}
}
