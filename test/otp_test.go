package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aph138/dekamond/internal/app"
)

var myApp *app.Application

func setup() {
	// logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// jwtKey, err := authentication.GenerateKey(8)
	// if err != nil {
	// 	logger.Error(fmt.Sprintf("err when generating key for jwt: %s", err.Error()))
	// 	os.Exit(1)
	// }
	// jwt, err := authentication.NewJWT(jwtKey)
	// if err != nil {
	// 	logger.Error(fmt.Sprintf("err when creating JWT instance: %s", err.Error()))
	// 	os.Exit(1)
	// }
	// db := &MockDB{
	// 	user: map[string]string{},
	// }
	// myApp = app.NewApplication(logger, jwt, otp.NewOTP(time.Minute*2), db)
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}
func TestInvalidPhoneNumber(t *testing.T) {
	reqBody := app.LoginRequest{
		Phone: "0999",
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

type MockDB struct {
	// map phone to id
	user map[string]string
}

func (m *MockDB) SaveUser(phone string) (string, error) {
	e, exist := m.user[phone]
	if exist {
		return e, nil
	} else {
		id := fmt.Sprint(len(m.user))
		m.user[phone] = id
		return id, nil
	}
}
