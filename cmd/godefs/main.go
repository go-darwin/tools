// SPDX-FileCopyrightText: 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build darwin && amd64 && gc
// +build darwin,amd64,gc

// Command godefs generate "go tool cgo -godefs" template file.
package main

import (
	"flag"
	"log"
	"os"

	"go-darwin.dev/tools/pkg/godefs"
)

var flagPackageName string

func init() {
	flag.StringVar(&flagPackageName, "package", "godefs", "package name of godefs input file.")
}

func main() {
	flag.Parse()

	if err := godefs.MakeGodefs(flagPackageName, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
