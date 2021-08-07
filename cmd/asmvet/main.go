// SPDX-FileCopyrightText: 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

// Command asmvet checks for correctness of Go assembly and some low-level operations.
//
// Standalone version of the assembly checks in go vet.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
)

func main() {
	unitchecker.Main(
		asmdecl.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		framepointer.Analyzer,
		unsafeptr.Analyzer,
	)
}
