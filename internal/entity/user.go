package entity

import "go.mongodb.org/mongo-driver/v2/bson"

// User defines user structure
type User struct {
	ID    bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Phone string        `json:"phone,omitempty" bson:"phone,omitempty"`
}
