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
	"strings"

	"github.com/muesli/ishell"
)

func travel(cwd Bucket, path string) (Bucket, error) {
	var err error
	parts := strings.Split(path, "/")
	for i := 0; err == nil && cwd != nil && i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}

		part := parts[i]
		if part == ".." {
			if cwd.Prev() != nil {
				cwd = cwd.Prev()
			}
		} else if part != "." {
			cwd, err = cwd.Cd(part)
		}
	}

	return cwd, err
}

func parseKeyPath(cwd Bucket, path string) (Bucket, string, error) {
	slashIndex := strings.LastIndex(path, "/")
	var key string
	var err error
	if slashIndex < 0 {
		key = path
	} else {
		key = path[slashIndex+1:]
		cwd, err = travel(cwd, path[:slashIndex])
	}
	return cwd, key, err
}

func lsCmd(c *ishell.Context) {
	target := cwd
	if len(c.Args) > 0 {
		var err error
		target, err = travel(target, c.Args[0])
		if err != nil {
			c.Err(err)
			return
		}
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
	target, key, err := parseKeyPath(cwd, c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	data, err = target.Get(key)
	if err != nil {
		c.Err(err)
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

	target, key, err := parseKeyPath(cwd, c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	c.Err(target.Put(key, c.Args[1]))
}

func cdCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		/* go to root */
		for cwd.Prev() != nil {
			cwd = cwd.Prev()
		}
	} else {
		b, err := travel(cwd, c.Args[0])
		if err != nil {
			c.Err(err)
			return
		}
		cwd = b
	}

	shell.SetPrompt(fmt.Sprintf(promptFmt, fname, cwd.String()))
}

func mkdirCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		c.Err(errors.New("mkdir: missing bucket name"))
		return
	}

	target, key, err := parseKeyPath(cwd, c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	c.Err(target.Mkdir(key))
}

func rmCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		c.Err(errors.New("rm: missing bucket or key name"))
		return
	}

	target, key, err := parseKeyPath(cwd, c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	c.Err(target.Rm(key))
}
