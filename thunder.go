/*
 * Thunder, BoltDB's interactive shell
 *     Copyright (c) 2017, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/boltdb/bolt"
	"github.com/chzyer/readline"
	"github.com/muesli/ishell"
)

var (
	shell *ishell.Shell
	cwd   Bucket

	promptFmt = "[%s %s] # "
	fname     string
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Printf("Usage: %v [db file]\n", os.Args[0])
		os.Exit(1)
	}

	fname = args[0]
	db, err := open(fname)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		cwd = NewRootBucket(tx)

		prompt := fmt.Sprintf(promptFmt, fname, cwd.String())
		shell = ishell.NewWithConfig(&readline.Config{Prompt: prompt})
		shell.Interrupt(interruptHandler)
		shell.EOF(eofHandler)
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

func eofHandler(c *ishell.Context) {
	os.Exit(0)
}

// extracts the last valid part of a Bucket key
// "/foo/ba" -> "/foo/"
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

func prefixBucket(s []string, name string) []string {
	for i, v := range s {
		s[i] = name + v
	}

	return s
}

func bucketCompleter(args []string, current string) []string {
	target, bucketName := partialBucketString(current)
	if target == nil {
		return []string{}
	}

	rval := printableList(target.Buckets(true))
	return prefixBucket(rval, bucketName)
}

func keyCompleter(args []string, current string) []string {
	target, bucketName := partialBucketString(current)
	if target == nil {
		return []string{}
	}

	rval := printableList(target.List())
	return prefixBucket(rval, bucketName)
}

func lsCmd(c *ishell.Context) {
	target := cwd
	if len(c.Args) > 0 {
		target = travel(target, c.Args[0])
	}

	if target == nil {
		c.Err(fmt.Errorf("'%s' is not a bucket", c.Args[0]))
		return
	}

	contents := target.List()
	entries := printableList(contents)
	for _, entry := range entries {
		c.Println(entry)
	}

	footnote := ""
	omitted := len(contents) - len(entries)
	if omitted > 0 {
		footnote = fmt.Sprintf(" (%d omitted in this list)", omitted)
	}
	c.Printf("%d keys in bucket%s\n", len(contents), footnote)
}

func getCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		c.Err(errors.New("get: missing key name"))
		return
	}

	var data []byte
	target, key := parseKeyPath(cwd, c.Args[0])
	if target != nil {
		data = target.Get(key)
	}
	if data == nil {
		c.Err(fmt.Errorf("No data at key '%s'", key))
		return
	}

	c.Println(string(data))
}

func putCmd(c *ishell.Context) {
	switch len(c.Args) {
	case 0:
		c.Err(errors.New("put: missing key name and value"))
		return
	case 1:
		c.Err(errors.New("put: missing value"))
		return
	}

	target, key := parseKeyPath(cwd, c.Args[0])
	if target == nil {
		c.Err(fmt.Errorf("Unable to set '%s' for key '%s'", c.Args[1], c.Args[0]))
		return
	}

	c.Err(target.Put(key, c.Args[1]))
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
			c.Err(fmt.Errorf("Unable to change to bucket '%s'", path))
			rval = cwd
		} else {
			rval = curr
		}
	}

	cwd = rval
	shell.SetPrompt(fmt.Sprintf(promptFmt, fname, cwd.String()))
}

func mkdirCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		c.Err(errors.New("mkdir: missing bucket name"))
		return
	}

	target, key := parseKeyPath(cwd, c.Args[0])
	if target == nil {
		c.Err(fmt.Errorf("Unable to create bucket at path '%v'", c.Args[0]))
		return
	}

	c.Err(target.Mkdir(key))
}

func rmCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		c.Err(errors.New("rm: missing bucket or key name"))
		return
	}

	target, key := parseKeyPath(cwd, c.Args[0])
	if target == nil {
		c.Err(fmt.Errorf("Unable to delete value at path '%v'", c.Args[0]))
		return
	}

	c.Err(target.Rm(key))
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

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsGraphic(r) {
			return false
		}
	}

	return true
}

func printableList(s []string) []string {
	r := []string{}
	for _, v := range s {
		if isPrintable(v) {
			r = append(r, v)
		}
	}

	return r
}
