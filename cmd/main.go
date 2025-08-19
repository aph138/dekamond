package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/aph138/dekamond/internal/app"
	"github.com/aph138/dekamond/internal/cache"
	"github.com/aph138/dekamond/internal/db"
	"github.com/aph138/dekamond/pkg/authentication"
	"github.com/kelseyhightower/envconfig"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Config struct {
	Port          int    `envconfig:"APP_PORT" default:"9000"`
	DBAddress     string `envconfig:"DB_ADDRESS" required:"true"`
	DBName        string `envconfig:"DB_NAME" required:"true"`
	DBUsername    string `envconfig:"DB_USERNAME"`
	DBPassword    string `envconfig:"DB_PASSWORD"`
	RedisAddress  string `envconfig:"REDIS_ADDRESS" require:"true"`
	RedisUsername string `envconfig:"REDIS_USERNAME"`
	RedisPassword string `envconfig:"REDIS_PASSWORD"`
	RedisDatabase int    `envconfig:"REDIS_PASSWORD" default:"0"`
}

func main() {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal("err when processing env variables", err.Error())
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	jwtKey, err := authentication.GenerateKey(8)
	if err != nil {
		logger.Error(fmt.Sprintf("err when generating key for jwt: %s", err.Error()))
		os.Exit(1)
	}
	jwt, err := authentication.NewJWT(jwtKey)
	if err != nil {
		logger.Error(fmt.Sprintf("err when creating JWT instance: %s", err.Error()))
		os.Exit(1)
	}
	dbAuthOpt := options.Client().
		SetAuth(options.Credential{Username: cfg.DBUsername, Password: cfg.DBPassword})
	db, err := db.NewMongo(cfg.DBAddress, cfg.DBName, time.Second*10, dbAuthOpt)
	if err != nil {
		logger.Error(fmt.Sprintf("err when creating MyMongo instance: %s", err.Error()))
		os.Exit(1)
	}
	myRedis, err := cache.NewRedis(&redis.Options{
		Addr:     cfg.RedisAddress,
		DB:       cfg.RedisDatabase,
		Username: cfg.RedisUsername,
		Password: cfg.RedisPassword,
	})
	if err != nil {
		logger.Error(fmt.Sprintf("err when creating MyRedis instance %s", err.Error()))
		os.Exit(1)
	}
	myApp := app.NewApplication(logger, jwt, myRedis, db)
	myApp.Run(cfg.Port)
}
