package http

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	maxRequestBodyBytes = 1 << 20

	zoneRequestSchema                     = "zone"
	updateZoneRequestSchema               = "updateZoneRequest"
	executeZoneRequestSchema              = "executeZoneRequest"
	fertilizerProgramRequestSchema        = "FertilizerProgram"
	weeklyProgramRequestSchema            = "WeeklyProgramRequest"
	temperatureProgramRequestSchema       = "TemperatureProgramRequest"
	updateTemperatureProgramRequestSchema = "UpdateTemperatureProgramRequest"
)

//go:embed openapi.yaml
var openAPISpec []byte

var loadRequestSchemas = sync.OnceValues(func() (openapi3.Schemas, error) {
	doc, err := openapi3.NewLoader().LoadFromData(openAPISpec)
	if err != nil {
		return nil, fmt.Errorf("failed to load openapi spec: %w", err)
	}
	return doc.Components.Schemas, nil
})

type InvalidRequestError struct {
	Reasons []string
}

func (i InvalidRequestError) Error() string {
	return "invalid request: " + strings.Join(i.Reasons, "; ")
}

func ReadRequest(w http.ResponseWriter, r *http.Request, schemaName string, req any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes))
	if err != nil {
		writeInvalidRequest(w, []string{err.Error()})
		return InvalidRequestError{Reasons: []string{err.Error()}}
	}
	if err = validateRequest(schemaName, body); err != nil {
		if invalid, ok := errors.AsType[InvalidRequestError](err); ok {
			writeInvalidRequest(w, invalid.Reasons)
			return err
		}
		WriteErrorResponse(w, http.StatusInternalServerError)
		return err
	}
	if err = json.Unmarshal(body, req); err != nil {
		writeInvalidRequest(w, []string{err.Error()})
		return InvalidRequestError{Reasons: []string{err.Error()}}
	}
	return nil
}

func validateRequest(schemaName string, body []byte) error {
	schemas, err := loadRequestSchemas()
	if err != nil {
		return err
	}
	schemaRef, ok := schemas[schemaName]
	if !ok || schemaRef.Value == nil {
		return fmt.Errorf("unknown request schema %q", schemaName)
	}
	var value any
	if err = json.Unmarshal(body, &value); err != nil {
		return InvalidRequestError{Reasons: []string{err.Error()}}
	}
	if err = schemaRef.Value.VisitJSON(value, openapi3.MultiErrors()); err != nil {
		return InvalidRequestError{Reasons: validationReasons(err)}
	}
	return nil
}

func validationReasons(err error) []string {
	var multi openapi3.MultiError
	if !errors.As(err, &multi) {
		return []string{schemaErrorReason(err)}
	}
	reasons := make([]string, 0, len(multi))
	for _, e := range multi {
		reasons = append(reasons, validationReasons(e)...)
	}
	return reasons
}

func schemaErrorReason(err error) string {
	var schemaErr *openapi3.SchemaError
	if !errors.As(err, &schemaErr) {
		return err.Error()
	}
	path := strings.Join(schemaErr.JSONPointer(), ".")
	if path == "" {
		return schemaErr.Reason
	}
	return fmt.Sprintf("%s: %s", path, schemaErr.Reason)
}

func writeInvalidRequest(w http.ResponseWriter, reasons []string) {
	errs := make([]Error, len(reasons))
	for i, reason := range reasons {
		errs[i] = Error{Code: ErrorCodeInvalidRequest, Message: reason}
	}
	WriteErrorResponse(w, http.StatusBadRequest, errs...)
}
