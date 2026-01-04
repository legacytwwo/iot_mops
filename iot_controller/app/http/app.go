package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
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

func (a *httpApp) GracefullyStop(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()

		if err := a.server.Shutdown(ctx); err != nil {
			log.Fatalf("failed to shutdown http server: %v", err)
		}
	}()
}
