package cache

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

// MyRedis implement Cache interface
type MyRedis struct {
	client *redis.Client
}

func NewRedis(opts *redis.Options) (*MyRedis, error) {
	c := redis.NewClient(opts)
	if cmd := c.Ping(context.Background()); cmd.Err() != nil {
		return nil, fmt.Errorf("err when connecting to redis %w", cmd.Err())
	}
	return &MyRedis{
		client: c,
	}, nil
}

func (r *MyRedis) NewOTPCode(phone string) (string, error) {
	otpKey := "otp:" + phone + ":login"

	// check if currently a valid code exists and return an error if it does
	exists, err := r.client.Exists(context.Background(), otpKey).Result()
	if err != nil {
		return "", fmt.Errorf("err when checking if the any code exists %w", err)
	}
	if exists > 0 {
		return "", ErrOTPStillValid
	}
	// generate a 6 digits random number
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("err when generating random OTP %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())

	_, err = r.client.Set(context.Background(), otpKey, code, time.Minute*2).Result()
	if err != nil {
		return "", fmt.Errorf("err when saving otp code %w", err)
	}

	return code, nil
}

func (r *MyRedis) VerifyOTPCode(phone string, code string) error {
	// using ZSET (sorted set) for implementing rate limit mechanism.
	key := "req:" + phone

	// remove any attempts longer than 10 minutes
	now := time.Now().Unix()
	_, err := r.client.ZRemRangeByScore(context.Background(),
		key,
		"0",
		fmt.Sprintf("%d", now-int64(time.Second*60*10)),
	).Result()

	if err != nil {
		return fmt.Errorf("err when ZRemRangeByScore %w", err)
	}

	// count all of the attempts in 10 minutes
	count, err := r.client.ZCard(context.Background(), key).Result()
	if err != nil {
		return fmt.Errorf("err when ZCard %w", err)
	}

	// return ErrRateLimit if attempts number is greater than 3
	if count >= 3 {
		return ErrRateLimit
	}

	// add new attempt
	_, err = r.client.ZAdd(context.Background(), key, redis.Z{Score: float64(now), Member: now}).Result()
	if err != nil {
		return fmt.Errorf("err when ZAdd %w", err)
	}

	// set an expire time in order to clean
	if _, err := r.client.Expire(context.Background(), key, time.Second*60*10).Result(); err != nil {
		return fmt.Errorf("err when Expire %w", err)
	}

	// get OTP code
	key = "otp:" + phone + ":login"
	expectedCode, err := r.client.Get(context.Background(), key).Result()

	// check if there is any code
	if err == redis.Nil {
		return ErrInvalidCode
	} else if err != nil {
		return fmt.Errorf("err when getting otp code %w", err)
	}

	if expectedCode != code {
		return ErrInvalidCode
	}

	// remove old valid code after successful login
	if _, err := r.client.Del(context.Background(), key).Result(); err != nil {
		return fmt.Errorf("err when deleting old otp code %w", err)
	}
	return nil
}
func (r *MyRedis) Close(ctx context.Context) error {
	return r.client.Close()
}
