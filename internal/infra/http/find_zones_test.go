package http_test

import (
	"context"
	"errors"
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/stretchr/testify/require"
)

func TestFindZones(t *testing.T) {
	id, name := "bbf", "bonsai big"
	zon := fixtures.ZoneBuilder{ID: &id, Name: &name, Relays: []zone.Relay{fixtures.RelayBuilder(1), fixtures.RelayBuilder(2)}}.Build()
	tests := []struct {
		name          string
		result        any
		qhErr         error
		expectedCode  int
		expectedZones []http.Zone
	}{
		{
			name:         "and query handler returns an error, then it returns an internal server error",
			qhErr:        errors.New(""),
			expectedCode: http2.StatusInternalServerError,
		},
		{
			name:          "and query handler returns no zones, then it returns an empty list",
			result:        []*zone.Zone{},
			expectedCode:  http2.StatusOK,
			expectedZones: []http.Zone{},
		},
		{
			name:         "and query handler returns zones, then it returns them with their relays",
			result:       []*zone.Zone{zon},
			expectedCode: http2.StatusOK,
			expectedZones: []http.Zone{
				{Id: id, Name: name, Relays: []http.ZoneRelays{1, 2}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FindZones http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			qh := &cqs.QueryHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Query) (any, error) {
					return tt.result, tt.qhErr
				},
			}
			handler := http.FindZones(qh, tracer())
			req := httptest.NewRequest(http2.MethodGet, "/zones", http2.NoBody)
			writer := httptest.NewRecorder()

			handler.ServeHTTP(writer, req)

			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
			if tt.expectedZones == nil {
				return
			}
			var zones []http.Zone
			readResponse(t, resp, &zones)
			require.Equal(t, tt.expectedZones, zones)
		})
	}
}
