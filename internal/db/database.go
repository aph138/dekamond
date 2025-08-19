package db

import (
	"context"
	"time"

	"github.com/aph138/dekamond/internal/entity"
)

type Database interface {
	// Close will close database
	Close(context.Context) error
	// SaveUser gets phone number and return either an error or user ID
	// If the user already exists, it only returns its ID
	SaveUser(string) (string, error)
	// FindUser(string) (*entity.User, error)
	SearchUser(...SearchUserOption) ([]entity.User, error)
}

type searchUserOption struct {
	phone        string
	registerFrom *time.Time
	registerTO   *time.Time
	pagination   searchUserPagination
}
type searchUserPagination struct {
	page  int64
	limit int64
}
type SearchUserOption func(*searchUserOption)

func SearchUserByPhone(phone string) SearchUserOption {
	return func(suo *searchUserOption) {
		suo.phone = phone
	}
}

func SearchUserByRegisterTime(from, to *time.Time) SearchUserOption {
	return func(suo *searchUserOption) {
		suo.registerFrom = from
		suo.registerTO = to
	}
}

// default values for pagination are: limit=10, page=1.
// any value less than 1 is treated as 1
func SearchUserByPagination(page, limit int64) SearchUserOption {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 1
	}
	return func(suo *searchUserOption) {
		suo.pagination.limit = limit
		suo.pagination.page = page
	}
}
