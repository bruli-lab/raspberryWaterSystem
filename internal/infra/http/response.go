package http

import (
	"encoding/json"
	"net/http"
)

func WriteResponse(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, errs ...Error) {
	var errResp ErrorResponseJson
	errResp.Errors = errs
	data, _ := json.Marshal(errResp)
	WriteResponse(w, statusCode, data)
}

type ErrorResponseJson struct {
	Errors []Error `json:"errors,omitempty"`
}

type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

const ErrorCodeInvalidRequest ErrorCode = "invalid_request"

type ErrorCode string
