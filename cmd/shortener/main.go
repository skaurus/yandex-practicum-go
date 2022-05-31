package main

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skaurus/yandex-practicum-go/internal/app"
	"github.com/skaurus/yandex-practicum-go/internal/config"
	"github.com/skaurus/yandex-practicum-go/internal/storage"

	"github.com/rs/zerolog/log"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	config := config.ParseConfig()
	store := storage.New(config)
	defer store.Close()

	router := app.SetupRouter(config, &store)
	srv := &http.Server{
		Addr:    config.ServerAddr,
		Handler: router,
	}
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("can't start the server")
		}
	}()

	sig := <-sigCh
	log.Info().Msgf("got signal %s, exiting\n", sig)
	close(sigCh)
	// когда сработает cancel - Shutdown выполнится принудительно, даже если
	// сервер ещё не дождался завершения всех запросов
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("can't shutdown the server")
	}

	log.Info().Msg("exited")
}
