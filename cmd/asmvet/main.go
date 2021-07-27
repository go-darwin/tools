// SPDX-FileCopyrightText: 2021 The go-darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

// Command asmvet checks for correctness of Go assembly.
//
// Standalone version of the assembly checks in go vet.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/framepointer"
)

func main() {
	unitchecker.Main(
		asmdecl.Analyzer,
		framepointer.Analyzer,
	)
}
