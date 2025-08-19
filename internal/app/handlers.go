package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aph138/dekamond/internal/cache"
	"github.com/aph138/dekamond/internal/db"
	"github.com/aph138/dekamond/internal/entity"
)

//	@Title			dekamond example swagger API
//	@Version		0.1
//	@Description	This an example OTP implementation
//
// @Host		localhost:9000
// @BasePath	/
type SearchResponse struct {
	Code   int           `json:"code"`
	Result []entity.User `json:"result"`
}
type LoginRequest struct {
	Phone string `json:"phone" example:"09012345678"`
}
type CheckRequest struct {
	Phone string `json:"phone" example:"09012345678"`
	Code  string `json:"code" example:"123456"`
}

// @Summery		Login endpoint
// @Description	Accepts a phone number and create an OTP code if the phone number is valid and no OTP code is currently valid that number.
// @Tags			login
// @Accept			json
// @Param			request	body	LoginRequest	true	"valid phone number as string"
// @Success		201		"No Content"
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

	code, err := a.cache.NewOTPCode(req.Phone)
	if err != nil {
		if errors.Is(err, cache.ErrOTPStillValid) {
			http.Error(w, "You still have a valid code. Please try again later.", http.StatusTooManyRequests)
			return
		} else {
			a.logger.Error(fmt.Sprintf("err when generating OTP code: %s", err.Error()))
			http.Error(w, "Something went wrong. Please contact support team.", http.StatusInternalServerError)
			return
		}
	}
	a.logger.Info(fmt.Sprintf("%s: %s", req.Phone, code))
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
}

// @Summery		check endpoint
// @Description	Accepts a phone number and an OTP code and return JWT token if they are valid
// @Tags			login
// @Accept			json
// @Produce		plain
// @Param			request	body		CheckRequest	true	"valid phone number and code"
// @Success		200		{string}	string			"JWT containing user ID"
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

	err := a.cache.VerifyOTPCode(req.Phone, req.Code)
	if err != nil {
		if errors.Is(err, cache.ErrRateLimit) {
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}
		http.Error(w, "invalid code", http.StatusUnauthorized)
		return
	}
	// will be saved in JWT payload
	var userID string

	// saving user in db if no records exist
	userID, err = a.db.SaveUser(req.Phone)
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
	w.Header().Add("Content-Type", "text/plain")
	fmt.Fprint(w, token)

}

// @Summery		Search for user
// @Description	Retrieve users
// @Produce		json
// @Tags			user
// @Param			phone		query		string	false	"A valid phone number for searching a specific user."																example(09012345678)
// @Param			register	query		string	false	"A date range to search for users who registered within that period in YYYY-MM-DD format, separated by a comma."	example(2024-01-01,2025-10-12)
// @Param			page		query		int		false	"The page number of the results. Default is 1. Negative numbers and zero are treated as 1."
// @Param			limit		query		int		false	"The number of items per page. Default is 10. Negative numbers and zero are treated as 1."
// @Success		200			{object}	SearchResponse
// @Router			/search [get]
func (a *Application) SearchUserHandler(w http.ResponseWriter, r *http.Request) {

	phoneQuery := r.URL.Query().Get("phone")
	registerQuery := r.URL.Query().Get("register")
	pageQuery := r.URL.Query().Get("page")
	limitQuery := r.URL.Query().Get("limit")

	// initialize user search option list
	opts := []db.SearchUserOption{}

	if len(phoneQuery) > 0 {
		//validate phone number
		rgx := regexp.MustCompile(`09\d{9}$`)
		if !rgx.MatchString(phoneQuery) {
			http.Error(w, "invalid phone number", http.StatusBadRequest)

			return
		}
		opts = append(opts, db.SearchUserByPhone(phoneQuery))
	}

	// a similar approach can be implemented for searching by latest login date and time
	// check if register has any value
	if len(registerQuery) > 0 {
		register := strings.Split(registerQuery, ",")
		if len(register) != 2 {
			http.Error(w, "register format must be YYYY-MM-DD,YYYY-MM,DD", http.StatusBadRequest)
			return
		}
		registerFrom, err := time.Parse("2006-01-02", register[0])
		if err != nil {
			http.Error(w, "invalid date format for register", http.StatusBadRequest)
			return
		}
		registerTo, err := time.Parse("2006-01-02", register[1])
		if err != nil {
			http.Error(w, "invalid date format for register", http.StatusBadRequest)
			return
		}
		opts = append(opts, db.SearchUserByRegisterTime(&registerFrom, &registerTo))
	}

	var limit int64 = 10
	var page int64 = 1
	var err error
	if len(limitQuery) > 0 {
		limit, err = strconv.ParseInt(limitQuery, 10, 64)
		if err != nil {
			http.Error(w, "invalid value for limit", http.StatusBadRequest)
			return
		}
	}
	if len(pageQuery) > 0 {
		page, err = strconv.ParseInt(pageQuery, 10, 64)
		if err != nil {
			http.Error(w, "invalid value for page", http.StatusBadRequest)
			return
		}
	}
	opts = append(opts, db.SearchUserByPagination(page, limit))

	list, err := a.db.SearchUser(opts...)
	if err != nil {
		a.logger.Error("err when searching user " + err.Error())
		http.Error(w, "something went wrong, try again later", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	res := SearchResponse{
		Code:   http.StatusOK,
		Result: list,
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		a.logger.Error("err when encoding search user result " + err.Error())
		http.Error(w, "something went wrong, try again later", http.StatusInternalServerError)
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
