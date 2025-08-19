package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/aph138/dekamond/docs"
	"github.com/aph138/dekamond/internal/cache"
	"github.com/aph138/dekamond/internal/db"
	"github.com/aph138/dekamond/pkg/authentication"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Application struct {
	logger *slog.Logger
	jwt    *authentication.JWT
	cache  cache.Cache
	db     db.Database
}

func NewApplication(
	logger *slog.Logger,
	jwt *authentication.JWT,
	cache cache.Cache,
	db db.Database,
) *Application {
	return &Application{
		logger: logger,
		jwt:    jwt,
		cache:  cache,
		db:     db,
	}
}

func (a *Application) Run(port int) {

	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", a.LoginHandler)
	mux.HandleFunc("POST /check", a.CheckHandler)
	mux.HandleFunc("GET /search", a.SearchUserHandler)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// at the production level, it's better to specify timeouts explicitly
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		a.logger.Info(fmt.Sprintf("starting server at %s", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("err when ListenAndServe %v\n", err)
		}
	}()

	// handling any interruption gracefully
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	a.logger.Info("starting graceful shutdown")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		a.logger.Error(fmt.Errorf("err when shutting down the server %w", err).Error())
	}
	// closing cache and database
	a.cache.Close(context.Background())
	a.db.Close(context.Background())

	a.logger.Info("server is successfully shut down")
}
