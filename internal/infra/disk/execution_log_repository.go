package disk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Log struct {
	Seconds    int       `json:"seconds"`
	ZoneName   string    `json:"zone_name"`
	ExecutedAt time.Time `json:"executed_at"`
}

type ExecutionLogRepository struct {
	path   string
	tracer trace.Tracer
}

func (e ExecutionLogRepository) Save(ctx context.Context, execLogs []program.ExecutionLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := e.tracer.Start(ctx, "ExecutionLogRepository.Save")
		defer span.End()
		logs := buildLogs(execLogs)
		if err := writeJsonFile(e.path, logs); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "execution logs saved")
		return nil
	}
}

func buildLogs(execLogs []program.ExecutionLog) []Log {
	logs := make([]Log, len(execLogs))
	for i, l := range execLogs {
		logs[i] = Log{
			Seconds:    l.Seconds().Int(),
			ZoneName:   l.ZoneName(),
			ExecutedAt: l.ExecutedAt(),
		}
	}
	return logs
}

func (e ExecutionLogRepository) FindAll(ctx context.Context) ([]program.ExecutionLog, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := e.tracer.Start(ctx, "ExecutionLogRepository.FindAll")
		defer span.End()
		logs, err := e.readLogs()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if errors.As(err, &CorruptExecutionLogError{}) {
				return []program.ExecutionLog{}, nil
			}
			return nil, err
		}
		span.SetStatus(codes.Ok, "execution logs found")
		return buildExecutionLogs(logs), nil
	}
}

func (e ExecutionLogRepository) readLogs() ([]Log, error) {
	data, err := os.ReadFile(e.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read file %s: %w", e.path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var logs []Log
	if err = json.Unmarshal(data, &logs); err != nil {
		return nil, e.quarantine(err)
	}
	return logs, nil
}

func (e ExecutionLogRepository) quarantine(cause error) error {
	corruptPath := fmt.Sprintf("%s%s%d", e.path, corruptFileSuffix, time.Now().UnixNano())
	if err := os.Rename(e.path, corruptPath); err != nil {
		return fmt.Errorf("failed to move corrupt execution log %s aside: %w", e.path, errors.Join(cause, err))
	}
	return CorruptExecutionLogError{movedTo: corruptPath, cause: cause}
}

const corruptFileSuffix = ".corrupt-"

type CorruptExecutionLogError struct {
	movedTo string
	cause   error
}

func (c CorruptExecutionLogError) Error() string {
	return fmt.Sprintf("corrupt execution log moved to %s: %s", c.movedTo, c.cause)
}

func buildExecutionLogs(logs []Log) []program.ExecutionLog {
	execLogs := make([]program.ExecutionLog, len(logs))
	for i, l := range logs {
		execLogs[i] = program.HydrateExecutionLog(program.HydrateSeconds(l.Seconds), l.ZoneName, l.ExecutedAt)
	}
	return execLogs
}

func NewExecutionLogRepository(path string, tracer trace.Tracer) ExecutionLogRepository {
	return ExecutionLogRepository{path: path, tracer: tracer}
}
