package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User defines user structure
type User struct {
	ID           bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Phone        string        `json:"phone,omitempty" bson:"phone,omitempty"`
	RegisteredAt time.Time     `json:"register_at,omitempty" bson:"register_at,omitempty"`
	LastLogin    time.Time     `json:"last_login,omitempty" bson:"last_login,omitempty"`
}
