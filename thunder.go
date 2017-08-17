/*
 * Thunder, BoltDB's interactive shell
 *     Copyright (c) 2017, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/chzyer/readline"
	"github.com/muesli/ishell"
)

var cwd Bucket

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Printf("Usage: %v [db file]\n", os.Args[0])
		os.Exit(1)
	}

	fname := args[0]
	db, err := open(fname)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		cwd = NewRootBucket(tx)

		prompt := fmt.Sprintf("[%s] # ", fname)
		shell := ishell.NewWithConfig(&readline.Config{Prompt: prompt})
		shell.Interrupt(interruptHandler)
		shell.SetHomeHistoryPath(".thunder_history")
		shell.Println("Thunder, Bolt's Interactive Shell")
		shell.Println("Type \"help\" for help.")
		shell.Println()

		shell.AddCmd(&ishell.Cmd{
			Name:      "ls",
			Func:      lsCmd,
			Help:      "list keys",
			LongHelp:  "lists keys in a bucket",
			Completer: bucketCompleter,
		})
		shell.AddCmd(&ishell.Cmd{
			Name:      "get",
			Func:      getCmd,
			Help:      "show value",
			LongHelp:  "shows the value of a key",
			Completer: keyCompleter,
		})
		shell.AddCmd(&ishell.Cmd{
			Name:      "put",
			Func:      putCmd,
			Help:      "put value",
			LongHelp:  "sets the value of a key",
			Completer: keyCompleter,
		})
		shell.AddCmd(&ishell.Cmd{
			Name:      "cd",
			Func:      cdCmd,
			Help:      "jump to a bucket",
			LongHelp:  "jumps to a bucket (empty to jump back to the root bucket)",
			Completer: bucketCompleter,
		})
		shell.AddCmd(&ishell.Cmd{
			Name:      "mkdir",
			Func:      mkdirCmd,
			Help:      "create a bucket",
			LongHelp:  "creates a bucket",
			Completer: keyCompleter,
		})
		shell.AddCmd(&ishell.Cmd{
			Name:      "rm",
			Func:      rmCmd,
			Help:      "delete a key",
			LongHelp:  "deletes a key",
			Completer: keyCompleter,
		})

		// when started with "exit" as first argument, assume non-interactive execution
		if len(os.Args) > 1 && os.Args[1] == "exit" {
			shell.Process(os.Args[2:]...)
		} else {
			// start shell
			shell.Run()
			// teardown
			shell.Close()
		}

		return nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func interruptHandler(c *ishell.Context, count int, line string) {
	if count >= 2 {
		c.Println("Interrupted")
		os.Exit(1)
	}
	c.Println("Press Ctrl-C once more to exit")
}

func partialBucketString(s string) (Bucket, string) {
	a := strings.Split(s, "/")
	if len(a) > 0 {
		a = a[:len(a)-1]
	}
	if len(a) > 0 {
		return travel(cwd, strings.Join(a, "/")), strings.Join(a, "/") + "/"
	}

	return cwd, ""
}

func bucketCompleter(args []string, current string) []string {
	target, bucketName := partialBucketString(current)
	if target == nil {
		return []string{}
	}

	rval := target.Buckets(true)
	for i, v := range rval {
		rval[i] = bucketName + v
	}
	return rval
}

func keyCompleter(args []string, current string) []string {
	target, bucketName := partialBucketString(current)
	if target == nil {
		return []string{}
	}

	rval := target.List()
	for i, v := range rval {
		rval[i] = bucketName + v
	}
	return rval
}

func lsCmd(c *ishell.Context) {
	target := cwd
	if len(c.Args) > 0 {
		target = travel(target, c.Args[0])
	}

	if target != nil {
		contents := target.List()
		for _, entry := range contents {
			c.Println(entry)
		}
	} else {
		c.Printf("'%s' is not a bucket\n", c.Args[0])
	}
}

func getCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		c.Println("Missing key")
	} else {
		target, key := parseKeyPath(cwd, c.Args[0])
		var data []byte
		if target != nil {
			data = target.Get(key)
		}
		if data == nil {
			c.Printf("No data at key %s\n", key)
		} else {
			fmt.Printf("%s\n", string(data))
		}
	}
}

func putCmd(c *ishell.Context) {
	if len(c.Args) < 2 {
		fmt.Println("Missing key or value")
	} else {
		target, key := parseKeyPath(cwd, c.Args[0])
		if target == nil {
			fmt.Printf("Unable to set '%s' for key '%s'\n", c.Args[1], c.Args[0])
		} else {
			target.Put(key, c.Args[1])
		}
	}
}

func cdCmd(c *ishell.Context) {
	var rval Bucket
	if len(c.Args) < 1 {
		/* go to root */
		parent := cwd.Prev()
		rval = cwd
		for parent != nil {
			rval = parent
			parent = rval.Prev()
		}
	} else {
		path := c.Args[0]
		curr := travel(cwd, path)
		if curr == nil {
			c.Printf("Unable to change to bucket '%s'\n", path)
			rval = cwd
		} else {
			rval = curr
		}
	}
	cwd = rval
}

func mkdirCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		fmt.Printf("Mkdir command must specify key\n")
	} else {
		target, key := parseKeyPath(cwd, c.Args[0])
		if target == nil {
			fmt.Printf("Unable to create bucket at path %v\n", c.Args[0])
		} else {
			target.Mkdir(key)
		}
	}
}

func rmCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		fmt.Printf("Rm command must specify key\n")
	} else {
		target, key := parseKeyPath(cwd, c.Args[0])
		if target == nil {
			fmt.Printf("Unable to delete value at path %v\n", c.Args[0])
		} else {
			target.Rm(key)
		}
	}
}

func travel(cwd Bucket, path string) Bucket {
	parts := strings.Split(path, "/")
	for i := 0; cwd != nil && i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}

		part := parts[i]
		if part == ".." {
			cwd = cwd.Prev()
		} else if part != "." {
			cwd = cwd.Cd(part)
		}
	}
	return cwd
}

func parseKeyPath(cwd Bucket, path string) (Bucket, string) {
	slashIndex := strings.LastIndex(path, "/")
	var key string
	if slashIndex < 0 {
		key = path
	} else {
		key = path[slashIndex+1:]
		cwd = travel(cwd, path[:slashIndex])
	}
	return cwd, key
}

func open(fname string) (*bolt.DB, error) {
	if _, err := os.Stat(fname); err != nil {
		return nil, fmt.Errorf("Unable to stat database file '%s': %v", fname, err)
	}
	db, err := bolt.Open(fname, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("Unable to open database file: '%s': %v", fname, err)
	}

	return db, nil
}
