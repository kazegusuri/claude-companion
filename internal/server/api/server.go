package api

import (
	"net/http"

	"github.com/kazegusuri/claude-companion/internal/server/db"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SetupEchoServer creates and configures an Echo server with API routes
func SetupEchoServer(database *db.DB) *echo.Echo {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Create the API handler
	apiHandler := CreateEchoStrictHandler(database)

	// Register API routes
	RegisterHandlers(e, apiHandler)

	return e
}

// CreateHTTPHandler creates an http.Handler from Echo server
func CreateHTTPHandler(database *db.DB) http.Handler {
	e := SetupEchoServer(database)
	return e
}
