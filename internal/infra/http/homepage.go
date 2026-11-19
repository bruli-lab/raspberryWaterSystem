package http

import (
	"net/http"
)

const VersionHeader = "X-App-Version"

func Homepage(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(VersionHeader, version)
		WriteResponse(w, http.StatusOK, nil)
	}
}
