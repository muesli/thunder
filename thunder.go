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
	c.Println("Press Ctrl-C once more to exit without saving the database")
}

func eofHandler(c *ishell.Context) {
	shell.Close()
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

// extracts the last valid part of a Bucket key
// "/foo/ba" -> "/foo/"
func partialBucketString(s string) (Bucket, string, error) {
	a := strings.Split(s, "/")
	if len(a) > 0 {
		a = a[:len(a)-1]
	}
	if len(a) > 0 {
		b, err := travel(cwd, strings.Join(a, "/"))
		if err != nil {
			return b, "", err
		}
		return b, strings.Join(a, "/") + "/", nil
	}

	return cwd, "", nil
}

func prefixBucket(s []string, name string) []string {
	for i, v := range s {
		s[i] = name + v
	}

	return s
}

func bucketCompleter(args []string, current string) []string {
	target, bucketName, err := partialBucketString(current)
	if err != nil {
		return []string{}
	}

	rval := printableList(target.Buckets(true))
	return prefixBucket(rval, bucketName)
}

func keyCompleter(args []string, current string) []string {
	target, bucketName, err := partialBucketString(current)
	if err != nil {
		return []string{}
	}

	rval := printableList(target.List())
	return prefixBucket(rval, bucketName)
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
