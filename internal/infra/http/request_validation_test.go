package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	http2 "github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestRequestValidation(t *testing.T) {
	tests := []struct {
		name             string
		method, route    string
		path, body       string
		handler          func(cqs.CommandHandler) http.HandlerFunc
		expectedCode     int
		expectedMessage  string
		expectedCommands int
	}{
		{
			name:            "an execute zone request without seconds, then it returns a bad request without executing the zone",
			method:          http.MethodPost,
			route:           "/zones/{id}/execute",
			path:            "/zones/zone-a/execute",
			body:            `{}`,
			handler:         func(ch cqs.CommandHandler) http.HandlerFunc { return http2.ExecuteZone(ch, tracer()) },
			expectedCode:    http.StatusBadRequest,
			expectedMessage: `property "seconds" is missing`,
		},
		{
			name:            "an execute fertilizer zone request with a string as seconds, then it returns a bad request",
			method:          http.MethodPost,
			route:           "/zones/fertilizer/{id}/execute",
			path:            "/zones/fertilizer/zone-a/execute",
			body:            `{"seconds":"20"}`,
			handler:         func(ch cqs.CommandHandler) http.HandlerFunc { return http2.ExecuteFertilizerZone(ch, tracer()) },
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "seconds",
		},
		{
			name:             "a create zone request with relay 4, then it creates the zone",
			method:           http.MethodPost,
			route:            "/zones",
			path:             "/zones",
			body:             `{"id":"zone-d","name":"zone d","relays":[4]}`,
			handler:          func(ch cqs.CommandHandler) http.HandlerFunc { return http2.CreateZone(ch, tracer()) },
			expectedCode:     http.StatusOK,
			expectedCommands: 1,
		},
		{
			name:            "a create zone request with a system relay, then it returns a bad request",
			method:          http.MethodPost,
			route:           "/zones",
			path:            "/zones",
			body:            `{"id":"zone-d","name":"zone d","relays":[5]}`,
			handler:         func(ch cqs.CommandHandler) http.HandlerFunc { return http2.CreateZone(ch, tracer()) },
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "relays",
		},
		{
			name:            "a create zone request without relays and name, then it returns a bad request with every reason",
			method:          http.MethodPost,
			route:           "/zones",
			path:            "/zones",
			body:            `{"id":"zone-d"}`,
			handler:         func(ch cqs.CommandHandler) http.HandlerFunc { return http2.CreateZone(ch, tracer()) },
			expectedCode:    http.StatusBadRequest,
			expectedMessage: `property "name" is missing`,
		},
		{
			name:   "a create daily program request without fertilizer, then it creates the program",
			method: http.MethodPost,
			route:  "/programs/daily",
			path:   "/programs/daily",
			body:   `{"hour":"10:00","executions":[{"seconds":20,"zones":["zone-a"]}]}`,
			handler: func(ch cqs.CommandHandler) http.HandlerFunc {
				return http2.CreateProgram(ch, http2.DailyProgram, tracer())
			},
			expectedCode:     http.StatusOK,
			expectedCommands: 1,
		},
		{
			name:   "a create daily program request without executions, then the domain rejects it with a bad request",
			method: http.MethodPost,
			route:  "/programs/daily",
			path:   "/programs/daily",
			body:   `{"hour":"10:00","executions":[]}`,
			handler: func(ch cqs.CommandHandler) http.HandlerFunc {
				return http2.CreateProgram(ch, http2.DailyProgram, tracer())
			},
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "empty programs",
		},
		{
			name:   "a create daily program request with seconds over the limit, then the domain rejects it with a bad request",
			method: http.MethodPost,
			route:  "/programs/daily",
			path:   "/programs/daily",
			body:   `{"hour":"10:00","executions":[{"seconds":121,"zones":["zone-a"]}]}`,
			handler: func(ch cqs.CommandHandler) http.HandlerFunc {
				return http2.CreateProgram(ch, http2.DailyProgram, tracer())
			},
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "seconds must be between 1 and 120",
		},
		{
			name:            "a create weekly program request with a program without zones, then the domain rejects it with a bad request",
			method:          http.MethodPost,
			route:           "/programs/weekly",
			path:            "/programs/weekly",
			body:            `{"week_day":"Monday","programs":[{"hour":"10:00","executions":[{"seconds":20,"zones":[]}]}]}`,
			handler:         func(ch cqs.CommandHandler) http.HandlerFunc { return http2.CreateWeeklyProgram(ch, tracer()) },
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "empty execution zones",
		},
		{
			name:         "a create temperature program request with an out of range temperature, then it returns a bad request instead of sending an empty program",
			method:       http.MethodPost,
			route:        "/programs/temperature",
			path:         "/programs/temperature",
			body:         `{"temperature":60,"programs":[{"hour":"10:00","executions":[{"seconds":20,"zones":["zone-a"]}]}]}`,
			handler:      func(ch cqs.CommandHandler) http.HandlerFunc { return http2.CreateTemperatureProgram(ch, tracer()) },
			expectedCode: http.StatusBadRequest,
		},
		{
			name:             "an update temperature program request with only programs, then it updates the program",
			method:           http.MethodPut,
			route:            "/programs/temperature/{temperature}",
			path:             "/programs/temperature/20",
			body:             `{"programs":[{"hour":"10:00","executions":[{"seconds":20,"zones":["zone-a"]}]}]}`,
			handler:          func(ch cqs.CommandHandler) http.HandlerFunc { return http2.UpdateTemperatureProgram(ch, tracer()) },
			expectedCode:     http.StatusOK,
			expectedCommands: 1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an http handler with request validation,
		when `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]cqs.Event, error) {
					return nil, nil
				},
			}
			server := chi.NewMux()
			server.Method(tt.method, tt.route, tt.handler(ch))
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			writer := httptest.NewRecorder()

			server.ServeHTTP(writer, req)

			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
			require.Len(t, ch.HandleCalls(), tt.expectedCommands)
			if tt.expectedMessage == "" {
				return
			}
			var body http2.ErrorResponseJson
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
			messages := make([]string, len(body.Errors))
			for i, e := range body.Errors {
				messages[i] = e.Message
			}
			require.Contains(t, strings.Join(messages, "\n"), tt.expectedMessage)
		})
	}
}

func TestCreateProgramFertilizerFlag(t *testing.T) {
	tests := []struct {
		name               string
		body               string
		expectedFertilizer bool
	}{
		{
			name:               "without fertilizer, then it creates a program without fertilizer",
			body:               `{"hour":"10:00","executions":[{"seconds":20,"zones":["zone-a"]}]}`,
			expectedFertilizer: false,
		},
		{
			name:               "with fertilizer true, then it creates a program with fertilizer",
			body:               `{"hour":"10:00","fertilizer":true,"executions":[{"seconds":20,"zones":["zone-a"]}]}`,
			expectedFertilizer: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a CreateProgram http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]cqs.Event, error) {
					return nil, nil
				},
			}
			handler := http2.CreateProgram(ch, http2.DailyProgram, tracer())
			req := httptest.NewRequest(http.MethodPost, "/programs/daily", strings.NewReader(tt.body))
			writer := httptest.NewRecorder()

			handler.ServeHTTP(writer, req)

			require.Equal(t, http.StatusOK, writer.Result().StatusCode)
			require.Len(t, ch.HandleCalls(), 1)
			cmd, ok := ch.HandleCalls()[0].Cmd.(app.CreateDailyProgramCommand)
			require.True(t, ok)
			require.Equal(t, tt.expectedFertilizer, cmd.Program.IsFertilizer())
		})
	}
}
