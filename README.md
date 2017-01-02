Thunder
=======

BoltDB's Interactive Shell

## Installation

Make sure you have a working Go environment (Go 1.4 or higher is required).
See the [install instructions](http://golang.org/doc/install.html).

To install Thunder, simply run:

    go get github.com/muesli/thunder

## Usage

```
$ thunder somebolt.db
Thunder, Bolt's Interactive Shell
Type "help" for help.

[somebolt.db /] #
```

### List keys in a bucket

```
[somebolt.db /] # ls
OneBucket/
AnotherBucket/
2 keys in bucket

[somebolt.db /] # ls OneBucket/
SubBucket/
SomeKey
2 keys in bucket
```

### Get the value of a key

```
[somebolt.db /] # get OneBucket/SomeKey
Much Value
```

### Set/change the value of a key
```
[somebolt.db /] # put OneBucket/SomeKey "Different Value"
```

### Delete a value or bucket
```
[somebolt.db /] # rm OneBucket/SomeKey
[somebolt.db /] # rm OneBucket/SubBucket
```

### Create a new bucket
```
[somebolt.db /] # mkdir AnotherBucket/NewBucket
```

### Change scope to a different bucket
```
[somebolt.db /] # cd AnotherBucket/NewBucket
```

## Development

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/muesli/thunder)
[![Build Status](https://travis-ci.org/muesli/thunder.svg?branch=master)](https://travis-ci.org/muesli/thunder)
[![Go ReportCard](http://goreportcard.com/badge/muesli/thunder)](http://goreportcard.com/report/muesli/thunder)
