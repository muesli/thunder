Thunder
=======

BoltDB's Interactive Shell

## Installation

Make sure you have a working Go environment. See the [install instructions](http://golang.org/doc/install.html).

To install Thunder, simply run:

    go get github.com/muesli/thunder

## Usage

```
$ thunder somebolt.db
Thunder, Bolt's Interactive Shell
Type "help" for help.

[somebolt.db] # ls
OneBucket/
AnotherBucket/

[somebolt.db] # ls OneBucket/
SubBucket/
SomeKey

[somebolt.db] # get OneBucket/SomeKey
Much Value

[somebolt.db] # put OneBucket/SomeKey "Different Value"
[somebolt.db] # rm OneBucket/SomeKey
[somebolt.db] # mkdir AnotherBucket/NewBucket
[somebolt.db] # cd AnotherBucket/NewBucket
[somebolt.db] # put NewKey "Newest Value"
...
```

## Development

API docs can be found [here](http://godoc.org/github.com/muesli/thunder).

[![Build Status](https://secure.travis-ci.org/muesli/thunder.png)](http://travis-ci.org/muesli/thunder)
[![Coverage Status](https://coveralls.io/repos/github/muesli/thunder/badge.svg?branch=master)](https://coveralls.io/github/muesli/thunder?branch=master)
[![Go ReportCard](http://goreportcard.com/badge/muesli/thunder)](http://goreportcard.com/report/muesli/thunder)
