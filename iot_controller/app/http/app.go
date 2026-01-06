package app

import (
	"context"
	"errors"
	"log"
	"net/http"
)

type httpApp struct {
	server *http.Server
}

func New(server *http.Server) *httpApp {
	return &httpApp{server: server}
}

func (a *httpApp) Run() {
	go func() {
		if err := a.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start http server: %v", err)
		}
	}()
}

func (a *httpApp) GracefullyStop(ctx context.Context) error {
	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
