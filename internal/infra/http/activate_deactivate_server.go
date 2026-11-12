package http

import (
	"errors"
	"net/http"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	ActivateAction   = "activate"
	DeactivateAction = "deactivate"
)

func ActivateDeactivateServer(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "ActivateDeactivateServerRequest")
		defer span.End()
		action := chi.URLParam(r, "action")
		var active bool
		switch action {
		case ActivateAction:
			active = true
		case DeactivateAction:
			active = false
		default:
			err := errors.New("invalid action name")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return
		}
		if _, err := ch.Handle(ctx, app.ActivateDeactivateServerCmd{Active: active}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusInternalServerError)
			return
		}
		span.SetStatus(codes.Ok, "server activated/deactivated")
		WriteResponse(w, http.StatusOK, nil)
	}
}
