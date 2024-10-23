package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jandiralceu/greenlight/internal/data"
	_ "github.com/lib/pq"
	"github.com/subosito/gotenv"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

type application struct {
	config config
	logger *slog.Logger
	models data.Models
}

func main() {
	_ = gotenv.Load()
	// Declare an instance of the config struct.
	var cfc config

	// Read the value of the port and env command-line flags into the config struct. We default to using
	// the port number 4000 and the environment "development" if no corresponding flags are provided.
	flag.IntVar(&cfc.port, "port", 4000, "API server port")
	flag.StringVar(&cfc.env, "env", "development", "Environment (development|staging|production)")

	// Read the DSN value from the db-dsn command-line flag into the config struct. We
	// default to using our development DSN if no flag is provided.
	flag.StringVar(&cfc.db.dsn, "db-dsn", os.Getenv("DB_CONNECTION_STRING"), "PostgreSQL DSN")

	// Read the connection pool settings from command-line flags into the config struct.
	// Notice that the default values we're using are the ones we discussed above?
	flag.IntVar(&cfc.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfc.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfc.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	// Create command line flags to read the setting values into the config struct.
	// Notice that we use true as the default for the 'enabled' setting?
	flag.Float64Var(&cfc.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfc.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfc.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Parse()

	// Initialize a new structured logger which writes log entries to the standard out stream.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Call the openDB() helper function (see below) to create the connection pool,
	// passing in the config struct. If this returns an error, we log it and exit the
	// application immediately.
	db, err := openDB(cfc)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Defer a call to db.Close() so that the connection pool is closed before the
	// main() function exits.
	defer db.Close()

	// Also log a message to say that the connection pool has been successfully
	// established.
	logger.Info("database connection pool established")

	// Declare an instance of the application struct, containing the config struct and the logger.
	app := &application{
		config: cfc,
		logger: logger,
		models: data.NewModels(db),
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
	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of open (in-use + idle) connections in the pool. Note that
	// passing a value less than or equal to 0 will mean there is no limit.
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Set the maximum idle timeout for connections in the pool. Passing a duration less
	// than or equal to 0 will mean that connections are not closed due to their idle time.
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
