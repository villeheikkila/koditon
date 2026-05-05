package api

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestAddRoutesBuildsSchemas(t *testing.T) {
	api := humago.New(http.NewServeMux(), NewConfig("Koditon API", "test"))
	a := API{logger: slog.Default()}
	a.AddRoutes(api)
}
