package cache

import (
	"context"
	"errors"
)

var ErrOTPStillValid = errors.New("a valid OTP still exists")
var ErrInvalidCode = errors.New("invalid OTP code")
var ErrRateLimit = errors.New("rate limit exceeded")

type Cache interface {
	// Close closes all connections and releases resources, if any exists.
	// Calling it ends the operations gracefully.
	// It uses context if it is possible.
	Close(context.Context) error

	// NewOTPCode takes an identifier and generate an OTP code if one doesn't exist.
	// It returns ErrOTPStillValid if a valid key still exist.
	NewOTPCode(string) (string, error)

	// Verify gets a phone number and an OTP code in order to verify the code.
	// It returns ErrRateLimit with must to wait time if user exceeds rate limit.
	// It returns ErrInvalidCode if the code doesn't exist or is wrong.
	VerifyOTPCode(string, string) error
}
