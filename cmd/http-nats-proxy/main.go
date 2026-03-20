// Package main is the main package.
package main

import (
	"context"
	"errors"
	"http-nats-proxy/api/restapi"
	"http-nats-proxy/internal/api"
	"log/slog"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/choria-io/fisk"
	"github.com/nats-io/nats.go"

	"github.com/oklog/run"
)

func main() {
	os.Exit(runmain())
}

// nolint:funlen
func runmain() int {
	var addr string
	var natsURL string
	var logLevel string
	var defaultTimeout time.Duration

	app := fisk.New("http-nats-proxy", "http-nats-proxy")
	app.Flag("addr", "Listen address").Default(":8080").StringVar(&addr)
	app.Flag("nats-url", "NATS server URL").Default("nats://localhost:4222").StringVar(&natsURL)
	app.Flag("log-level", "Log level").Default("info").StringVar(&logLevel)
	app.Flag("default-timeout", "Default timeout for waiting NATS responses").Default("5s").
		DurationVar(&defaultTimeout)

	_, err := app.Parse(os.Args[1:])
	if err != nil {
		slog.Error("failed to parse arguments: ", "err", err)

		return 1
	}
	var lvl slog.Level
	err = lvl.UnmarshalText([]byte(logLevel))
	if err != nil {
		slog.Error("failed to parse log level", "err", err)

		return 1
	}
	slog.SetLogLoggerLevel(lvl)

	g := run.Group{}

	slog.Info("connecting to nats server...", "url", natsURL)
	ncConn, err := nats.Connect(natsURL)
	if err != nil {
		slog.Error("failed to connect to NATS", "err", err)

		return 1
	}
	srv := api.NewServer(ncConn, api.WithDefaultTimeout(defaultTimeout))

	var opts []restapi.ServerOption
	server, err := restapi.NewServer(srv, opts...)
	if err != nil {
		slog.Error("failed to initialize server", "err", err)

		return 1
	}
	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))
	httpServer := http.Server{
		Addr:              addr,
		Handler:           server,
		ReadHeaderTimeout: 5 * time.Second, //nolint:mnd
	}

	g.Add(func() error {
		slog.Info("starting server...", "addr", addr)
		err := httpServer.ListenAndServe()
		if err != nil {
			return err
		}

		return nil
	}, func(_ error) {
		err := httpServer.Shutdown(context.Background())
		if err != nil {
			slog.Warn("failed to shutdown http server", "err", err)
		}
	})
	err = g.Run()
	if err != nil && !errors.Is(err, &run.SignalError{}) {
		slog.Error("failed to start server", "err", err)

		return 1
	}
	slog.Info("DONE")

	return 0
}
