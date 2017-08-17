/*
 * Thunder, BoltDB's interactive shell
 *     Copyright (c) 2017, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

type Bucket interface {
	// Prev returns the parent of this Bucket.
	// Returns nil if this Bucket is root.
	Prev() Bucket

	// Cd changes the current Bucket to the bucket stored under key.
	Cd(key string) Bucket

	// List returns keys for all values and buckets in this bucket.
	// Bucket keys are suffixed with a slash.
	List() []string

	// Bucket returns keys for all sub-buckets in this bucket.
	// Bucket keys are suffixed with a slash if withTrailingSlash is true.
	Buckets(withTrailingSlash bool) []string

	// Get returns a value for a key or nil if none found.
	Get(key string) []byte

	// Put stores a value at the given key.
	Put(key, value string)

	// Mkdir creates a new bucket with the given key.
	Mkdir(key string)

	// Rm removes a bucket or value with the given key.
	Rm(key string)

	// Returns the full path of the bucket
	String() string
}

type RootBucket struct {
	tx *bolt.Tx
}

func NewRootBucket(tx *bolt.Tx) *RootBucket {
	return &RootBucket{tx}
}

func (rl *RootBucket) Prev() Bucket {
	return nil
}

func (rl *RootBucket) Cd(key string) Bucket {
	var rval Bucket
	nested := rl.tx.Bucket([]byte(key))
	if nested != nil {
		rval = &SubBucket{nested, "/" + key, rl}
	}
	return rval
}

func (rl *RootBucket) List() []string {
	curr := rl.tx.Cursor()
	return list(curr)
}

func (rl *RootBucket) Buckets(withTrailingSlash bool) []string {
	curr := rl.tx.Cursor()
	return buckets(curr, withTrailingSlash)
}

func (rl *RootBucket) Get(key string) []byte {
	return nil
}

func (rl *RootBucket) Put(key, value string) {
	fmt.Println("Cannot store values at root Bucket")
}

func (rl *RootBucket) Mkdir(key string) {
	_, err := rl.tx.CreateBucket([]byte(key))
	if err != nil {
		fmt.Printf("Unable to create bucket at key '%v': %v\n", key, err)
	}
}

func (rl *RootBucket) Rm(key string) {
	err := rl.tx.DeleteBucket([]byte(key))
	if err != nil {
		fmt.Printf("Unable to delete bucket at key '%v': %v\n", key, err)
	}
}

func (rl *RootBucket) String() string {
	return "/"
}

type SubBucket struct {
	b    *bolt.Bucket
	path string
	prev Bucket
}

func (bl *SubBucket) Prev() Bucket {
	return bl.prev
}

func (bl *SubBucket) Cd(key string) Bucket {
	var rval Bucket
	nested := bl.b.Bucket([]byte(key))
	if nested != nil {
		rval = &SubBucket{nested, bl.path + "/" + key, bl}
	}
	return rval
}

func (bl *SubBucket) List() []string {
	curr := bl.b.Cursor()
	return list(curr)
}

func (bl *SubBucket) Buckets(withTrailingSlash bool) []string {
	curr := bl.b.Cursor()
	return buckets(curr, withTrailingSlash)
}

func (bl *SubBucket) Get(key string) []byte {
	return bl.b.Get([]byte(key))
}

func (bl *SubBucket) Put(key, value string) {
	err := bl.b.Put([]byte(key), []byte(value))
	if err != nil {
		fmt.Printf("Unable to store '%v' at '%v': %v\n", value, key, err)
	}
}

func (bl *SubBucket) Mkdir(key string) {
	_, err := bl.b.CreateBucket([]byte(key))
	if err != nil {
		fmt.Printf("Unable to create bucket at key '%v': %v\n", key, err)
	}
}

func (bl *SubBucket) Rm(key string) {
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
		fmt.Printf("Unable to delete '%v': %v\n", key, err)
	}
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
