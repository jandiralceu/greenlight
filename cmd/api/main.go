package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
}

type application struct {
	config config
	logger *slog.Logger
}

func main() {
	// Declare an instance of the config struct.
	var cfc config

	// Read the value of the port and env command-line flags into the config struct. We default to using
	// the port number 4000 and the environment "development" if no corresponding flags are provided.
	flag.IntVar(&cfc.port, "port", 4000, "API server port")
	flag.StringVar(&cfc.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	// Initialize a new structured logger which writes log entries to the standard out stream.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Declare an instance of the application struct, containing the config struct and the logger.
	app := &application{
		config: cfc,
		logger: logger,
	}

	// Use the httprouter instance returned by app.routes() as the server handler.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfc.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Start the HTTP server.
	logger.Info("starting server", "addr", srv.Addr, "env", cfc.env)
	err := srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}
