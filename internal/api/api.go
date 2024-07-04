package api

import (
	"net/http"

	v1 "github.com/emilekm/demos-hub/internal/api/v1"
)

const (
	v1Path = "/api/v1/"
)

func Routes(servers *v1.Servers) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", servers.Routes()))

	return mux
}
