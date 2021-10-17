// Copyright 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"errors"
	"go/format"
	"io"
	"os"
	"regexp"
)

func fix(r io.Reader) int {
	data, err := io.ReadAll(r)
	if err != nil && !errors.Is(err, io.EOF) {
		log.Error(err, "read r")
		return exitFailure
	}

	// fix struct field names first letter is digit
	reStartDigit := regexp.MustCompile(`(?m)^(\t+)(\d+([\t\w_]+)?)`)
	out := reStartDigit.ReplaceAll(data, []byte(`${1}X_${2}`))

	// remove cgo padding fields
	rePaddingFields := regexp.MustCompile(`Pad_cgo_\d+`)
	out = rePaddingFields.ReplaceAll(out, []byte("_"))

	// remove padding, hidden, or unused fields
	reUnused := regexp.MustCompile(`Padding`)
	out = reUnused.ReplaceAll(out, []byte("_"))

	// replace 'void' C type to Go uintptr type
	out = bytes.ReplaceAll(out, []byte("_Ctype_void"), []byte("uintptr"))

	// trimc godefs generate based files directory name
	cwd, _ := os.Getwd()
	out = bytes.ReplaceAll(out, []byte(cwd+string(os.PathSeparator)), nil)

	out, err = format.Source(out)
	if err != nil {
		return exitFailure
	}
	os.Stdout.Write(out)

	return exitSuccess
}
