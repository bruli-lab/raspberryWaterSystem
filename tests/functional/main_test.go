//go:build functional

package functional_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/tests/harness"
)

var appBinary string

func TestMain(m *testing.M) {
	binary, cleanup, err := harness.BuildApp(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	appBinary = binary
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func newZone() http.Zone {
	zo := fixtures.ZoneBuilder{}.Build()
	relays := make([]http.ZoneRelays, len(zo.Relays()))
	for i, r := range zo.Relays() {
		relays[i] = http.ZoneRelays(r.Id().Int())
	}
	return http.Zone{Id: zo.Id(), Name: zo.Name(), Relays: relays}
}
