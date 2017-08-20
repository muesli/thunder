/*
 * Thunder, BoltDB's interactive shell
 *     Copyright (c) 2017, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

var (
	// ErrNoSuchBucket gets returned when accessing a non-existing bucket
	ErrNoSuchBucket = errors.New("no such bucket")
	// ErrNoKeysAtRoot gets returned when accessing a root-bucket key
	ErrNoKeysAtRoot = errors.New("cannot store values at root bucket")
	// ErrNoSuchKey gets returned when accessing a non-existing key
	ErrNoSuchKey = errors.New("no such key")
)

// Bucket is an interface to Bolt's buckets
type Bucket interface {
	// Prev returns the parent of this Bucket.
	// Returns nil if this Bucket is root.
	Prev() Bucket

	// Cd changes the current Bucket to the bucket stored under key.
	Cd(key string) (Bucket, error)

	// List returns keys for all values and buckets in this bucket.
	// Bucket keys are suffixed with a slash.
	List() []string

	// Bucket returns keys for all sub-buckets in this bucket.
	// Bucket keys are suffixed with a slash if withTrailingSlash is true.
	Buckets(withTrailingSlash bool) []string

	// Get returns a value for a key.
	Get(key string) ([]byte, error)

	// Put stores a value at the given key.
	Put(key, value string) error

	// Mkdir creates a new bucket with the given key.
	Mkdir(key string) error

	// Rm removes a bucket or value with the given key.
	Rm(key string) error

	// Returns the full path of the bucket.
	String() string
}

// RootBucket represents Bolt's root bucket, which can store other buckets
// but not regular values
type RootBucket struct {
	tx *bolt.Tx
}

// NewRootBucket returns a new RootBucket
func NewRootBucket(tx *bolt.Tx) *RootBucket {
	return &RootBucket{tx}
}

// Prev returns nil as a RootBucket has no parents
func (rl *RootBucket) Prev() Bucket {
	return nil
}

// Cd changes scope to another bucket
func (rl *RootBucket) Cd(key string) (Bucket, error) {
	b := rl.tx.Bucket([]byte(key))
	if b == nil {
		return rl, ErrNoSuchBucket
	}
	return &SubBucket{b, "/" + key, rl}, nil
}

// List returns all keys in this bucket
func (rl *RootBucket) List() []string {
	c := rl.tx.Cursor()
	return list(c)
}

// Buckets returns all sub-buckets in this bucket
func (rl *RootBucket) Buckets(withTrailingSlash bool) []string {
	c := rl.tx.Cursor()
	return buckets(c, withTrailingSlash)
}

// Get returns the value of a key
func (rl *RootBucket) Get(key string) ([]byte, error) {
	return nil, ErrNoKeysAtRoot
}

// Put sets the value of a key
func (rl *RootBucket) Put(key, value string) error {
	return ErrNoKeysAtRoot
}

// Mkdir creates a new sub-bucket
func (rl *RootBucket) Mkdir(key string) error {
	_, err := rl.tx.CreateBucket([]byte(key))
	if err != nil {
		return fmt.Errorf("unable to create bucket at key '%v': %v", key, err)
	}
	return nil
}

// Rm deletes a key
func (rl *RootBucket) Rm(key string) error {
	err := rl.tx.DeleteBucket([]byte(key))
	if err != nil {
		return fmt.Errorf("unable to delete bucket at key '%v': %v", key, err)
	}
	return nil
}

func (rl *RootBucket) String() string {
	return "/"
}

// SubBucket represents a Bolt bucket
type SubBucket struct {
	b    *bolt.Bucket
	path string
	prev Bucket
}

// Prev returns the parent bucket
func (bl *SubBucket) Prev() Bucket {
	return bl.prev
}

// Cd changes scope to another bucket
func (bl *SubBucket) Cd(key string) (Bucket, error) {
	b := bl.b.Bucket([]byte(key))
	if b == nil {
		if _, err := bl.Get(key); err == nil {
			return bl, errors.New("key is a value, not a bucket")
		}
		return bl, ErrNoSuchBucket
	}
	return &SubBucket{b, bl.path + "/" + key, bl}, nil
}

// List returns all keys in this bucket
func (bl *SubBucket) List() []string {
	curr := bl.b.Cursor()
	return list(curr)
}

// Buckets returns all sub-buckets in this bucket
func (bl *SubBucket) Buckets(withTrailingSlash bool) []string {
	curr := bl.b.Cursor()
	return buckets(curr, withTrailingSlash)
}

// Get returns the value of a key
func (bl *SubBucket) Get(key string) ([]byte, error) {
	b := bl.b.Get([]byte(key))

	if b == nil {
		bucket := bl.b.Bucket([]byte(key))
		if bucket != nil {
			return nil, errors.New("key is a bucket, not a value")
		}
		return nil, ErrNoSuchKey
	}

	return b, nil
}

// Put sets the value of a key
func (bl *SubBucket) Put(key, value string) error {
	err := bl.b.Put([]byte(key), []byte(value))
	if err != nil {
		return fmt.Errorf("unable to store '%v' at '%v': %v", value, key, err)
	}
	return nil
}

// Mkdir creates a new sub-bucket
func (bl *SubBucket) Mkdir(key string) error {
	_, err := bl.b.CreateBucket([]byte(key))
	if err != nil {
		return fmt.Errorf("unable to create bucket at key '%v': %v", key, err)
	}
	return nil
}

// Rm deletes a key
func (bl *SubBucket) Rm(key string) error {
	keyBytes := []byte(key)
	c := bl.b.Cursor()
	k, v := c.Seek(keyBytes)
	err := fmt.Errorf("no such key")
	if k != nil && string(k) == key {
		if v == nil {
			err = bl.b.DeleteBucket(keyBytes)
		} else {
			err = bl.b.Delete(keyBytes)
		}
	}

	if err != nil {
		return fmt.Errorf("unable to delete '%v': %v", key, err)
	}
	return nil
}

func (bl *SubBucket) String() string {
	return bl.path
}

func list(curr *bolt.Cursor) []string {
	var rval []string
	for k, v := curr.First(); k != nil; k, v = curr.Next() {
		val := string(k)
		if v == nil {
			rval = append(rval, val+"/")
		} else {
			rval = append(rval, val)
		}
	}
	return rval
}

func buckets(curr *bolt.Cursor, withTrailingSlash bool) []string {
	var rval []string
	for k, v := curr.First(); k != nil; k, v = curr.Next() {
		if v != nil {
			continue
		}
		bk := string(k)
		if withTrailingSlash {
			bk += "/"
		}
		rval = append(rval, bk)
	}
	return rval
}
