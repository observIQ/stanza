//go:generate mockery --name=^(Database)$ --output=../testutil --outpkg=testutil --case=snake

package database

import (
	"time"

	"go.etcd.io/bbolt"
)

// OpenDatabase will open a connection to a bbolt db for the passed in file
func OpenDatabase(file string) (*bbolt.DB, error) {
	options := &bbolt.Options{Timeout: 1 * time.Second}
	return bbolt.Open(file, 0600, options)
}
