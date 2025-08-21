package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aph138/dekamond/internal/app"
	"github.com/aph138/dekamond/internal/cache"
	"github.com/aph138/dekamond/internal/db"
	"github.com/aph138/dekamond/pkg/authentication"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	DBUsername = "test_user"
	DBPassword = "test_password"
)

var (
	redisEndpoint string
	mongoEndpoint string
)

var myApp *app.Application
var containers = []testcontainers.Container{}

func clean() {
	for _, c := range containers {
		testcontainers.TerminateContainer(c)
	}
	containers = containers[:0]
}

func setup_redis() (testcontainers.Container, error) {
	rc, err := redisContainer.Run(context.Background(), "redis:latest")
	if err != nil {
		return nil, fmt.Errorf("err when running redis container %w", err)
	}
	re, err := rc.Endpoint(context.Background(), "")
	if err != nil {
		return nil, fmt.Errorf("err when getting redis endpoint %w", err)
	}
	redisEndpoint = re
	containers = append(containers, rc)
	return rc, nil
}
func setup_mongo() (testcontainers.Container, error) {
	mc, err := mongodb.Run(
		context.Background(),
		"mongo:8.0.11",
		mongodb.WithUsername(DBUsername),
		mongodb.WithPassword(DBPassword),
	)
	if err != nil {
		return nil, fmt.Errorf("err when running mongodb container %w", err)
	}
	me, err := mc.Endpoint(context.Background(), "")
	if err != nil {
		return nil, fmt.Errorf("err when getting mongo endpoint %w", err)
	}
	mongoEndpoint = me
	containers = append(containers, mc)
	return mc, nil
}
func setup() error {
	_, err := setup_redis()
	if err != nil {
		return fmt.Errorf("err when setting redis up %w", err)
	}
	_, err = setup_mongo()
	if err != nil {
		return fmt.Errorf("err when setting mongo up %w", err)
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
	dbOpt := options.Client().SetAuth(options.Credential{
		Username: DBUsername, Password: DBPassword,
	})
	db, err := db.NewMongo("mongodb://"+mongoEndpoint, "dekamond_test", time.Second*15, dbOpt)
	if err != nil {
		log.Fatalln("err when connecting to db server", err.Error())
	}
	myRedis, err := cache.NewRedis(&redis.Options{
		Addr: redisEndpoint,
		DB:   1,
	})
	if err != nil {
		log.Fatalln("err when connecting to redis", err.Error())
	}
	myApp = app.NewApplication(logger, jwt, myRedis, db)
	return nil
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	clean()
	os.Exit(code)
}
func TestInvalidPhoneNumber(t *testing.T) {
	reqBody := app.LoginRequest{
		Phone: "09991",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	myApp.CheckHandler(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 status code but got %d", res.StatusCode)
	}

}
func TestRedisOTP(t *testing.T) {
	myRedis, err := cache.NewRedis(&redis.Options{Addr: redisEndpoint})
	if err != nil {
		t.Error(err)
	}
	phone := "09012345678"

	// generate new code
	code, err := myRedis.NewOTPCode(phone)
	if err != nil {
		t.Errorf("err when generating new otp code %s", err.Error())
	}

	// generate new code when currently a valid code exists
	_, err = myRedis.NewOTPCode(phone)
	if !errors.Is(err, cache.ErrOTPStillValid) {
		t.Fatalf("expected %s but got %s", err.Error(), err.Error())
	}

	// generate a invalid code
	var invalidCode string
	for {
		r := rand.Intn(999999)
		c := fmt.Sprintf("%06d", r)
		if c != code {
			invalidCode = c
			break
		}
	}

	// check for invalid code
	err = myRedis.VerifyOTPCode(phone, invalidCode)
	if !errors.Is(err, cache.ErrInvalidCode) {
		t.Fatalf("expected %s but got %s", cache.ErrInvalidCode, err.Error())
	}

	// check for valid code
	err = myRedis.VerifyOTPCode(phone, code)
	if err != nil {
		t.Fatalf("didn't expected any error but got %s", err.Error())
	}

	// check for old valid code
	err = myRedis.VerifyOTPCode(phone, code)
	if !errors.Is(err, cache.ErrInvalidCode) {
		t.Fatalf("expected %s but got %s", cache.ErrInvalidCode, err.Error())
	}

	// check for non-existing phone number
	err = myRedis.VerifyOTPCode("09098765432", code)
	if !errors.Is(err, cache.ErrInvalidCode) {
		t.Fatalf("expected %s but got %s", cache.ErrInvalidCode, err.Error())
	}

}
