package main

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"net/http"
	"os"
	"os/signal"
	"syscall"

	ihttp "github.com/rafimuhammad01/tracking-app/http"
	"github.com/rafimuhammad01/tracking-app/track"
)

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go HTTPServer(done, &wg)
	wg.Wait()
}

func HTTPServer(done chan os.Signal, wg *sync.WaitGroup) {
	port := ":8080"

	hub := track.NewHub()
	handler := ihttp.NewHandler(hub)

	srv := &http.Server{
		Addr: port,
	}

	http.HandleFunc("/location", handler.GetLatestLocation)

	go func() {
		log.Info().Any("port", port).Msg("starting http server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start http server")
			return
		}
	}()

	<-done // graceful shutdown
	log.Info().Msg("stopping http server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to shutdown server")
	}

	log.Info().Msg("http server stopped")
	wg.Done()
}
