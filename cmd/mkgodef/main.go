// Copyright 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	exitSuccess = iota
	exitFailure
)

const (
	fnamePackage      = "package"
	fnameMode         = "mode"
	fnameHeader       = "header"
	fnameArg          = "arg"
	fnameSource       = "source"
	fnamegIgnoreMacro = "ignore-macro"
	fnameConfig       = "config"
	fnameDebug        = "debug"
)

var (
	flagPackage      string
	flagMode         []string
	flagHeaders      []string
	flagArgs         []string
	flagSources      []string
	flagIgnoreMacros []string
	flagConfig       string
	flagDebug        bool
)

var log logr.Logger

func main() {
	flag.StringVar(&flagPackage, fnamePackage, "", "package name of godef")
	flag.StringSliceVar(&flagMode, fnameMode, []string{"enum, func, type"}, "generate mode")
	flag.StringSliceVar(&flagHeaders, fnameHeader, nil, "C header to analyze")
	flag.StringSliceVar(&flagArgs, fnameArg, nil, "arg for analyze")
	flag.StringSliceVar(&flagSources, fnameSource, nil, "additional C source to analyze")
	flag.StringSliceVar(&flagIgnoreMacros, fnamegIgnoreMacro, nil, "ignore macro names")
	flag.StringVar(&flagConfig, fnameConfig, "", "config file to analyze")
	flag.BoolVar(&flagDebug, fnameDebug, false, "debug log output")
	flag.Parse()

	lvl := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	zl, err := zap.NewDevelopment(zap.IncreaseLevel(lvl), zap.AddCaller(), zap.AddStacktrace(zapcore.DebugLevel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "new zap development logger: %v\n", err)
		os.Exit(int(exitFailure))
	}
	if flagDebug {
		lvl.SetLevel(zapcore.DebugLevel)
	}
	log = zapr.NewLogger(zl)

	cmd := flag.Arg(0)
	if cmd == "fix" {
		os.Exit(fix(os.Stdin))
	}

	os.Exit(run(flag.CommandLine))
}
