package otp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"
)

type Code struct {
	code      string
	createdAt time.Time
}
type OTP struct {
	data map[string]Code
	ttl  time.Duration
	mu   sync.RWMutex // for concurrent safety
}

// I'm using a simple map for convenience and faster access.
// But if horizontal scaling were necessary, an in-memory
// database like redis or a dedicated service would be a better option.
func NewOTP(t time.Duration) *OTP {
	otp := &OTP{
		data: make(map[string]Code),
		ttl:  t,
		mu:   sync.RWMutex{},
	}
	go otp.cleanup()
	return otp
}
func (o *OTP) cleanup() {
	for {
		time.Sleep(time.Minute * 5)
		o.mu.Lock()
		for key, code := range o.data {
			if time.Since(code.createdAt) >= o.ttl {
				delete(o.data, key)
			}
		}
		o.mu.Unlock()
	}
}

var ErrOTPStillValid = errors.New("a valid OTP still exists")
var ErrRateLimit = errors.New("")

// NewCode generate and save an OTP code if it doesn't exist.
// It returns ErrOTPStillValid if there is a valid key.
// It returns ErrRateLimit with must to wait time if user exceeds rate limit.
func (o *OTP) NewCode(key string) error {
	// check if a valid code doesn't exist already
	o.mu.RLock()
	entity, exist := o.data[key]
	o.mu.RUnlock()

	if exist && time.Since(entity.createdAt) < o.ttl {
		return ErrOTPStillValid
	}

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return fmt.Errorf("err when generating random OTP %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())
	log.Printf("phone: %s, code: %s", key, code)
	o.mu.Lock()
	o.data[key] = Code{
		code:      code,
		createdAt: time.Now(),
	}
	o.mu.Unlock()
	return nil
}

// Verify verifies a code for a phone number.
// It returns false if it doesn't exist or is code is wrong.
func (o *OTP) Verify(key, code string) bool {
	o.mu.RLock()
	otp, exist := o.data[key]
	o.mu.RUnlock()

	// check for any valid code
	if !exist || time.Since(otp.createdAt) >= o.ttl || otp.code != code {
		return false
	}

	o.mu.Lock()
	delete(o.data, key)
	o.mu.Unlock()

	return true
}
