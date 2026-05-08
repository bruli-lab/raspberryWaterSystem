//go:build functional

package functional

import (
	"fmt"
	http2 "net/http"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/stretchr/testify/require"
)

func TestZones(t *testing.T) {
	var savedZone *zone.Zone
	t.Run(`Given a Create zone endpoint`, func(t *testing.T) {
		t.Run(`when a request is sent without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			zo := fixtures.ZoneBuilder{}.Build()
			relays := make([]int, len(zo.Relays()))
			for i, r := range zo.Relays() {
				relays[i] = r.Id().Int()
			}
			req := http.CreateZoneRequestJson{
				Id:     zo.Id(),
				Name:   zo.Name(),
				Relays: relays,
			}
			resp, err := buildRequestAndSend(ctx, req, nil, http2.MethodPost, "/zones", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`when a request is sent with authorization,
		then it returns an ok`, func(t *testing.T) {
			zo := fixtures.ZoneBuilder{}.Build()
			relays := make([]int, len(zo.Relays()))
			for i, r := range zo.Relays() {
				relays[i] = r.Id().Int()
			}
			req := http.CreateZoneRequestJson{
				Id:     zo.Id(),
				Name:   zo.Name(),
				Relays: relays,
			}
			resp, err := buildRequestAndSend(ctx, req, authorizationHeader(), http2.MethodPost, "/zones", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
			savedZone = zo
		})
	})
	t.Run(`Given a find zone endpoint`, func(t *testing.T) {
		t.Run(`when a request without authorization is sent
		then it returns an unauthorized`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, nil, http2.MethodGet, "/zones", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`when a request is sent with authorization,
		then it returns a valid zones list`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, authorizationHeader(), http2.MethodGet, "/zones", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
			var schema []http.ZonesItemResponse
			readResponse(t, resp, &schema)
		})
	})
	t.Run(`Given an execute zone endpoint,
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			url := fmt.Sprintf("/zones/%s/execute", savedZone.Id())
			resp, err := buildRequestAndSend(ctx, nil, nil, http2.MethodPost, url, cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
		then it returns an ok`, func(t *testing.T) {
			url := fmt.Sprintf("/zones/%s/execute", savedZone.Id())
			req := http.ExecuteZoneRequestJson{Seconds: 5}
			resp, err := buildRequestAndSend(ctx, req, authorizationHeader(), http2.MethodPost, url, cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
	t.Run(`Given an execute fertilizer zone endpoint,
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			url := fmt.Sprintf("/zones/fertilizer/%s/execute", savedZone.Id())
			resp, err := buildRequestAndSend(ctx, nil, nil, http2.MethodPost, url, cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
		then it returns an ok`, func(t *testing.T) {
			url := fmt.Sprintf("/zones/fertilizer/%s/execute", savedZone.Id())
			req := http.ExecuteZoneRequestJson{Seconds: 5}
			resp, err := buildRequestAndSend(ctx, req, authorizationHeader(), http2.MethodPost, url, cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
	t.Run(`Given an Update zone endpoint`, func(t *testing.T) {
		t.Run(`when a request is sent without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			zo := fixtures.ZoneBuilder{}.Build()
			relays := make([]int, len(zo.Relays()))
			for i, r := range zo.Relays() {
				relays[i] = r.Id().Int()
			}
			req := http.UpdateZoneRequestJson{
				Name:   zo.Name(),
				Relays: relays,
			}
			url := fmt.Sprintf("/zones/%s", savedZone.Id())
			resp, err := buildRequestAndSend(ctx, req, nil, http2.MethodPut, url, cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`when a request is sent with authorization,
		then it returns an ok`, func(t *testing.T) {
			zo := fixtures.ZoneBuilder{}.Build()
			relays := make([]int, len(zo.Relays()))
			for i, r := range zo.Relays() {
				relays[i] = r.Id().Int()
			}
			req := http.CreateZoneRequestJson{
				Id:     zo.Id(),
				Name:   zo.Name(),
				Relays: relays,
			}
			url := fmt.Sprintf("/zones/%s", savedZone.Id())
			resp, err := buildRequestAndSend(ctx, req, authorizationHeader(), http2.MethodPut, url, cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
			savedZone = zo
		})
	})
	t.Run(`Given a Remove zone endpoint,
	when a request is sent `, func(t *testing.T) {
		t.Run(`without authentication,
		then it return unauthorized`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, nil, http2.MethodDelete, "/zones/bbf", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authentication,
		then it returns ok`, func(t *testing.T) {
			zo := fixtures.ZoneBuilder{}.Build()
			relays := make([]int, len(zo.Relays()))
			for i, r := range zo.Relays() {
				relays[i] = r.Id().Int()
			}
			req := http.CreateZoneRequestJson{
				Id:     zo.Id(),
				Name:   zo.Name(),
				Relays: relays,
			}
			resp, err := buildRequestAndSend(ctx, req, authorizationHeader(), http2.MethodPost, "/zones", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
			resp, err = buildRequestAndSend(ctx, nil, authorizationHeader(), http2.MethodDelete, "/zones/"+zo.Id(), cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
}
