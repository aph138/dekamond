package db

type Database interface {
	// SaveUser gets phone number and return either an error or user ID
	// If the user already exists, it only returns its ID
	SaveUser(string) (string, error)
}
