package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aph138/dekamond/pkg/otp"
)

//	@title			dekamond example swagger API
//	@version		0.1
//	@description	This an example OTP implementation

// @host		localhost:9000
// @BasePath	/
type APIResponse struct {
	Code   int `json:"code"`
	Result any `json:"result"`
}
type LoginRequest struct {
	Phone string `json:"phone" example:"09012345678"`
}
type CheckRequest struct {
	Phone string `json:"phone" example:"09012345678"`
	Code  string `json:"code" example:"123456"`
}

// @Summery		Login endpoint
// @Description	accepts a phone number and create a OTP code if phone number is valid and there is no valid code for that number
// @Accept			json
// @Produce		json
// @Param			request	body		LoginRequest	true "valid phone number as string"
// @Success		200		{object}	APIResponse
// @Router			/login [post]
func (a *Application) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	reqDecoder := json.NewDecoder(r.Body)
	reqDecoder.DisallowUnknownFields() // for strict validation
	if err := reqDecoder.Decode(&req); err != nil {
		a.logger.Error(fmt.Sprintf("err when decoding body at /login: %s", err.Error()))
		http.Error(w, "something went wrong, try again", http.StatusInternalServerError)
		return
	}

	// validate phone number
	rgx := regexp.MustCompile(`09\d{9}$`)
	if !rgx.MatchString(req.Phone) {
		http.Error(w, "invalid phone number", http.StatusBadRequest)
		return
	}

	if err := a.otp.NewCode(req.Phone); err != nil {
		if errors.Is(err, otp.ErrOTPStillValid) {
			res := APIResponse{
				Code:   http.StatusOK,
				Result: "you still have a valid code. Please try again later",
			}
			if err := json.NewEncoder(w).Encode(res); err != nil {
				a.logger.Error(fmt.Sprintf("err when encoding response: %s", err.Error()))
				http.Error(w, "something went wrong, try again", http.StatusInternalServerError)
				return
			}
			return
		} else {
			a.logger.Error(fmt.Sprintf("err when generating OTP code: %s", err.Error()))
			http.Error(w, "something went wrong, try again", http.StatusInternalServerError)
			return
		}
	}
	res := APIResponse{
		Code:   http.StatusCreated,
		Result: "ok",
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		a.logger.Error(fmt.Sprintf("err when encoding response: %s", err.Error()))
		http.Error(w, "something went wrong, try again", http.StatusInternalServerError)
		return
	}
}

// @Summery		check endpoint
// @Description	accepts a phone number and an OTP code and return JWT token if they are valid
// @Accept			json
// @Produce		json
// @Param			request	body		CheckRequest	true "valid phone number and code"
// @Success		200		{object}	APIResponse
// @Router			/check [post]
func (a *Application) CheckHandler(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	reqDecoder := json.NewDecoder(r.Body)
	reqDecoder.DisallowUnknownFields() // for strict validation
	if err := reqDecoder.Decode(&req); err != nil {
		a.logger.Error(fmt.Sprintf("err when decoding body at /check: %s", err.Error()))
		http.Error(w, "something went wrong, try again", http.StatusInternalServerError)
		return
	}

	// validate phone number
	rgx := regexp.MustCompile(`09\d{9}$`)
	if !rgx.MatchString(req.Phone) {
		http.Error(w, "invalid phone number", http.StatusBadRequest)
		return
	}

	optResult := a.otp.Verify(req.Phone, req.Code)
	if optResult {
		// will be saved in JWT payload
		var userID string

		// saving user in db if no records exist
		userID, err := a.db.SaveUser(req.Phone)
		if err != nil {
			a.logger.Error(fmt.Sprintf("err when saving user at /check: %s", err.Error()))
			http.Error(w, "something went wrong, try again later", http.StatusInternalServerError)
			return
		}

		// generate JWT token
		// consider encrypting userID in real world scenario
		token, err := a.jwt.NewToken(map[string]string{"id": userID}, time.Hour*24)
		if err != nil {
			a.logger.Error(fmt.Sprintf("err when generating new JWT token: %s", err.Error()))
			http.Error(w, "something went wrong, try again later", http.StatusInternalServerError)
			return
		}
		res := APIResponse{
			Code:   http.StatusOK,
			Result: token,
		}
		if err := json.NewEncoder(w).Encode(res); err != nil {
			a.logger.Error(fmt.Sprintf("err when encoding JWT token: %s", err.Error()))
			http.Error(w, "something went wrong, try again later", http.StatusInternalServerError)
			return
		}

	} else {
		http.Error(w, "invalid code", http.StatusUnauthorized)
		return
	}

}

func (a *Application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" || !strings.HasPrefix(token, "Bearer ") {
			http.Error(w, "unauthorized access", http.StatusUnauthorized)
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")

		result, err := a.jwt.Parse(token)
		if err != nil {
			http.Error(w, "unauthorized access", http.StatusUnauthorized)
			return
		}
		_ = result["id"]
	})
}
