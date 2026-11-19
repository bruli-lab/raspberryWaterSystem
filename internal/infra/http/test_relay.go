package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestRelay(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "TestRelayRequest")
		defer span.End()
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusBadRequest)
			return
		}
		if _, err = ch.Handle(ctx, app.TestRelayCmd{RelayID: id}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.Is(err, zone.ErrUnknownRelay):
				WriteErrorResponse(w, http.StatusBadRequest)
			case errors.As(err, &app.CleanInProgressError{}):
				WriteErrorResponse(w, http.StatusConflict)
			default:
				WriteErrorResponse(w, http.StatusInternalServerError)
			}
			return
		}
		span.SetStatus(codes.Ok, "relay tested")
		WriteResponse(w, http.StatusOK, nil)
	}
}
