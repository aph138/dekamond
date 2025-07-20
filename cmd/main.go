package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aph138/dekamond/internal/app"
	"github.com/aph138/dekamond/internal/db"
	"github.com/aph138/dekamond/pkg/authentication"
	"github.com/aph138/dekamond/pkg/otp"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
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
	dbAuthOpt := options.Client().SetAuth(options.Credential{Username: "admin_user", Password: "password"})
	db, err := db.NewMongo("mongodb://db:27017/", "dekamond", time.Second*10, dbAuthOpt)
	if err != nil {
		logger.Error(fmt.Sprintf("err when creating MyMongo instance: %s", err.Error()))
		os.Exit(1)
	}
	myApp := app.NewApplication(logger, jwt, otp.NewOTP(time.Minute*2), db)
	myApp.Run()
}
