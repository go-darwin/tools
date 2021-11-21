// Copyright 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-clang/clang-v13/clang"
	yaml "github.com/goccy/go-yaml"
	flag "github.com/spf13/pflag"
)

func init() {
	spew.Config = spew.ConfigState{
		Indent:                  "  ",
		MaxDepth:                0, // maximum number of levels to descend into nested data structures.
		DisableMethods:          false,
		DisablePointerMethods:   false,
		DisablePointerAddresses: false, // don't spew the addresses of pointers
		DisableCapacities:       false, // don't spew capacities of collections
		ContinueOnMethod:        true,  // recursion should continue once a custom error or Stringer interface is invoked
		SortKeys:                true,  // maps should be spewed in a deterministic order
		SpewKeys:                true,  // if unable to sort map keys then spew keys to strings and sort those
	}
}

var ignoreMacros = map[string]bool{
	"__GNUC__":  true,
	"__APPLE__": true,
}

type Mode uint

const (
	EnumMode = 1 << iota
	TypeMode
	FuncMode
	RawFuncMode
)

// Config represents a mkgodef config.
type Config struct {
	Package      string            `yaml:"package,omitempty"`
	Mode         []string          `yaml:"mode,omitempty"`
	Headers      []string          `yaml:"header,omitempty"`
	Args         []string          `yaml:"args,omitempty"`
	Sources      []string          `yaml:"source,omitempty"`
	IgnoreMacros []string          `yaml:"ignoreMacro,omitempty"`
	GodefsMap    map[string]string `yaml:"godefsMap,omitempty"`
}

// ReadConfig reads config and return new Config from r.
func ReadConfig(r io.Reader) (config *Config, err error) {
	dec := yaml.NewDecoder(r)
	config = new(Config)
	if err := dec.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func ConfigFromFlags(flags *flag.FlagSet) (config *Config, err error) {
	configFile, err := flags.GetString(fnameConfig)
	if err != nil {
		return nil, err
	}
	if configFile != "" {
		f, err := os.Open(configFile)
		if err != nil {
			return nil, fmt.Errorf("open %s config file: %w", configFile, err)
		}
		defer f.Close()

		config, err = ReadConfig(f)
		if err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}

		return config, nil
	}

	pkgName, err := flags.GetString(fnamePackage)
	if err != nil {
		return nil, err
	}
	mode, err := flags.GetStringSlice(fnameMode)
	if err != nil {
		return nil, err
	}
	headers, err := flags.GetStringSlice(fnameHeader)
	if err != nil {
		return nil, err
	}
	args, err := flags.GetStringSlice(fnameSource)
	if err != nil {
		return nil, err
	}
	sources, err := flags.GetStringSlice(fnameSource)
	if err != nil {
		return nil, err
	}
	ignoreMacros, err := flags.GetStringSlice(fnamegIgnoreMacro)
	if err != nil {
		return nil, err
	}

	return &Config{
		Package:      pkgName,
		Mode:         mode,
		Args:         args,
		Headers:      headers,
		Sources:      sources,
		IgnoreMacros: ignoreMacros,
	}, nil
}

func run(flags *flag.FlagSet) int {
	config, err := ConfigFromFlags(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse configs: %v\n", err)
		return exitFailure
	}

	if config.Headers == nil {
		flag.Usage()
		fmt.Fprintf(os.Stderr, "please provide a file name to analyze\n")
		return exitFailure
	}

	if len(config.IgnoreMacros) > 0 {
		for _, macro := range config.IgnoreMacros {
			ignoreMacros[macro] = true
		}
	}

	idx := clang.NewIndex(1, 0)
	defer idx.Dispose()

	clangFlags := uint32(
		clang.TranslationUnit_DetailedPreprocessingRecord |
			clang.TranslationUnit_Incomplete |
			clang.TranslationUnit_PrecompiledPreamble |
			clang.TranslationUnit_ForSerialization |
			clang.TranslationUnit_CXXChainedPCH |
			clang.TranslationUnit_CreatePreambleOnFirstParse |
			clang.TranslationUnit_KeepGoing |
			clang.TranslationUnit_IncludeAttributedTypes,
	)

	funcMap := make(map[string]clang.Cursor)
	typeMap := make(map[clang.CursorKind][]clang.Cursor)
	enumMap := make(map[clang.Cursor][]clang.Cursor)

	for i := 0; i < len(config.Headers); i++ {
		header := config.Headers[i]

		tu := idx.ParseTranslationUnit(header, config.Args, nil, clangFlags)
		defer tu.Dispose()

		cursor := tu.TranslationUnitCursor()
		cursor.Visit(func(cursor, parent clang.Cursor) clang.ChildVisitResult {
			if cursor.IsNull() {
				return clang.ChildVisit_Continue
			}

			file, _, _, _ := cursor.Location().FileLocation()
			if !strings.Contains(file.Name(), filepath.Dir(header)) {
				log.V(1).Info("ignore file", "file", file.Name())
				return clang.ChildVisit_Continue
			}

			var skip bool
			switch kind := cursor.Kind(); kind {
			case clang.Cursor_FunctionDecl: // function
				funcMap[cursor.Spelling()] = cursor

				return clang.ChildVisit_Recurse

			case clang.Cursor_EnumDecl: // enum type
				enumMap[cursor] = []clang.Cursor{}

				return clang.ChildVisit_Recurse

			case clang.Cursor_EnumConstantDecl: // enum constant
				enumMap[parent] = append(enumMap[parent], cursor)

				return clang.ChildVisit_Recurse

			case clang.Cursor_StructDecl: // struct type
				typeMap[kind] = append(typeMap[kind], cursor)

				return clang.ChildVisit_Recurse

			case clang.Cursor_VarDecl: // variable type
				typeMap[kind] = append(typeMap[kind], cursor)

				return clang.ChildVisit_Recurse

			case clang.Cursor_UnionDecl:
				typeMap[kind] = append(typeMap[kind], cursor)

				return clang.ChildVisit_Recurse

			case clang.Cursor_TypedefDecl: // type
				typeMap[kind] = append(typeMap[kind], cursor)

				return clang.ChildVisit_Recurse

			case clang.Cursor_MacroDefinition: // skip include guard
				skip = true

				return clang.ChildVisit_Continue

			case clang.Cursor_MacroExpansion: // const
				defer func() { skip = false }() // deferred set false
				if skip {
					return clang.ChildVisit_Continue
				}

				name := cursor.DisplayName()
				if ignoreMacros[name] {
					return clang.ChildVisit_Continue
				}

				typeMap[kind] = append(typeMap[kind], cursor)

				return clang.ChildVisit_Recurse

			case clang.Cursor_VisibilityAttr, // skip
				clang.Cursor_InclusionDirective,
				clang.Cursor_ParmDecl,
				clang.Cursor_TypeRef,
				clang.Cursor_DeclRefExpr,
				clang.Cursor_PackedAttr,
				clang.Cursor_FieldDecl:

				return clang.ChildVisit_Continue

			default:
				typeMap[kind] = append(typeMap[kind], cursor)

				return clang.ChildVisit_Continue
			}
		})
	}

	var buf bytes.Buffer
	bio := bufio.NewWriter(&buf)
	var buf2 bytes.Buffer
	bio2 := bufio.NewWriter(&buf2)

	p(bio, "// Code generated by github.com/go-darwin/tools/cmd/mkgodef; DO NOT EDIT.\n")
	p(bio, "// Input to cgo -godefs.\n\n")
	p(bio, "//go:build ignore\n// +build ignore\n\n")

	if len(config.GodefsMap) > 0 {
		for goName, cName := range config.GodefsMap {
			p(bio, "// +godefs map %s %s\n", goName, cName)
		}
		p(bio, "\n")
	}

	p(bio, "package %s\n\n", config.Package)

	var mode Mode
	for _, m := range config.Mode {
		switch m {
		case "enum":
			mode |= EnumMode
		case "type":
			mode |= TypeMode
		case "func":
			mode |= FuncMode
		case "rawfunc":
			mode |= RawFuncMode
		}
	}

	if mode&EnumMode != 0 || mode&TypeMode != 0 {
		p(bio, "/*\n")
		for _, header := range config.Headers {
			p(bio, "#include <%s>\n", header)
		}
		if config.Sources != nil {
			for _, source := range config.Sources {
				p(bio, "%s\n", source)
			}
		}
		p(bio, "*/\n")
		p(bio, "import %q\n\n", "C")
	}

	seen := make(map[string]bool)
	if mode&EnumMode != 0 {
		// sort enumMap by DisplayName
		cursors := make([]clang.Cursor, len(enumMap))
		i := 0
		for cursor := range enumMap {
			cursors[i] = cursor
			i++
		}
		// sort cursors by DisplayName
		sort.Slice(cursors, func(i, j int) bool { return cursors[i].DisplayName() < cursors[j].DisplayName() })

		// sort enumMap by Kind
		for parent := range enumMap {
			sort.Slice(enumMap[parent], func(i, j int) bool { return enumMap[parent][i].Kind() < enumMap[parent][j].Kind() })
		}

		for _, cursor := range cursors {
			parent := cursor
			curs := enumMap[cursor]

			parentDisplayName := strings.TrimSuffix(parent.DisplayName(), "\n")
			parentName := upperCamelCase(parentDisplayName)
			if seen[parentName] {
				log.V(1).Info("ignore", "parent name", parentName)
				continue
			}
			seen[parentName] = true
			p(bio, "type %s C.enum_%s\n\n", parentName, parentDisplayName)

			p(bio, "const (\n")

			for _, cur := range curs {
				curDisplayName := strings.TrimSuffix(cur.DisplayName(), "\n")
				if seen[curDisplayName] {
					log.V(1).Info("ignore", "name", curDisplayName)
					continue
				}
				seen[curDisplayName] = true

				str := fmt.Sprintf("%[1]s %[2]s = C.%[1]s\n", curDisplayName, parentName)
				p(bio, "\t%s", str)
			}

			p(bio, ")\n\n")
		}
	}

	if mode&TypeMode != 0 {
		writeFn := func(format, goName, cName string, seen map[string]bool) bool {
			if seen[goName] {
				log.V(1).Info("ignore", "goName", goName, "cName", cName)
				return false
			}
			seen[goName] = true

			bio.WriteString(fmt.Sprintf(format, goName, cName))
			return true
		}

		// sort typeMap kind key by Spelling
		kinds := make([]clang.CursorKind, len(typeMap))
		i := 0
		for kind := range typeMap {
			kinds[i] = kind
			i++
		}
		sort.Slice(kinds, func(i, j int) bool { return kinds[i].Spelling() < kinds[j].Spelling() })

		for _, kind := range kinds {
			cursors := typeMap[kind]

			// sort cursors by DisplayName
			sort.Slice(cursors, func(i, j int) bool { return cursors[i].DisplayName() < cursors[j].DisplayName() })

			switch kind {
			case clang.Cursor_VarDecl:
				for _, cursor := range cursors {
					cName := cursor.DisplayName()
					if cName == "" {
						continue
					}
					goName := strings.TrimSuffix(upperCamelCase(cName), "T")

					if !writeFn("var %s = C.%s\n\n", goName, cName, seen) {
						continue
					}
				}

			case clang.Cursor_UnionDecl:
				for _, cursor := range cursors {
					cName := cursor.DisplayName()
					if cName == "" {
						continue
					}
					goName := upperCamelCase(cName)

					if !writeFn("type %s C.union_%s\n\n", goName, cName, seen) {
						continue
					}
				}

			case clang.Cursor_MacroExpansion:
				for _, cursor := range cursors {
					cName := cursor.DisplayName()
					if cName == "" {
						continue
					}
					goName := export(cName)

					if !writeFn("const %s = C.%s\n\n", goName, cName, seen) {
						continue
					}
				}

			case clang.Cursor_StructDecl:
				for _, cursor := range cursors {
					cName := cursor.DisplayName()
					if cName == "" {
						continue
					}
					goName := upperCamelCase(cName)

					if !writeFn("type %s C.struct_%s\n\n", goName, cName, seen) {
						continue
					}
				}

			case clang.Cursor_TypedefDecl:
				for _, cursor := range cursors {
					cName := cursor.DisplayName()
					if cName == "" {
						continue
					}
					goName := upperCamelCase(cName)

					if !writeFn("type %s C.%s\n\n", goName, cName, seen) {
						continue
					}
				}

			default:
				for _, cursor := range cursors {
					cName := cursor.DisplayName()
					if cName == "" {
						continue
					}
					goName := upperCamelCase(cName)
					bio2.WriteString(fmt.Sprintf("kind: %s, cName: %s, goName: %s\n", cursor.Kind(), cName, goName))
				}
			}
		}
	}

	if mode&FuncMode != 0 {
		// sort funcMap by DisplayName
		fns := make([]string, len(funcMap))
		i := 0
		for fn := range funcMap {
			fns[i] = fn
			i++
		}
		sort.Strings(fns)

		var sb strings.Builder
		seenFn := make(map[string]bool)
		for _, fn := range fns {
			cursor := funcMap[fn]
			if seenFn[fn] {
				log.V(1).Info("ignore", "s", fn, "kind", cursor.Kind())
				continue
			}
			seenFn[fn] = true

			switch cursor.Kind() {
			case clang.Cursor_FunctionDecl:
				p(&sb, "//sys func %s(", upperCamelCase(cursor.Spelling()))

				numArgs := cursor.NumArguments()
				for i := int32(0); i < numArgs; i++ {
					argName := strings.TrimSpace(lowerCamelCase(cursor.Argument(uint32(i)).DisplayName()))
					argType := convertGoType(cursor.Argument(uint32(i)).Type().CanonicalType().Spelling())

					p(&sb, "%s %s", argName, argType)
					if i+1 < numArgs {
						p(&sb, ", ")
					}
				}
				p(&sb, ") %s\n", convertGoType(cursor.ResultType().Spelling()))

				bio.WriteString(sb.String())
				sb.Reset()
			}
		}
	}

	if mode&RawFuncMode != 0 {
		// sort funcMap by DisplayName
		fns := make([]string, len(funcMap))
		i := 0
		for fn := range funcMap {
			fns[i] = fn
			i++
		}
		sort.Strings(fns)

		var sb strings.Builder
		seenFn := make(map[string]bool)
		for _, fn := range fns {
			cursor := funcMap[fn]
			if seenFn[fn] {
				log.V(1).Info("ignore", "s", fn, "kind", cursor.Kind())
				continue
			}
			seenFn[fn] = true

			switch cursor.Kind() {
			case clang.Cursor_FunctionDecl:
				p(&sb, "//sys func %s(", cursor.Spelling())

				numArgs := cursor.NumArguments()
				for i := int32(0); i < numArgs; i++ {
					argName := cursor.Argument(uint32(i)).DisplayName()
					argType := cursor.Argument(uint32(i)).Type().CanonicalType().Spelling()

					p(&sb, "%s %s", argName, argType)
					if i+1 < numArgs {
						p(&sb, ", ")
					}
				}
				p(&sb, ") %s\n", cursor.ResultType().Spelling())

				bio.WriteString(sb.String())
				sb.Reset()
			}
		}
	}

	bio.Flush()
	bio2.Flush()

	io.Copy(os.Stdout, &buf)
	os.Stdout.Sync()

	io.Copy(os.Stderr, &buf2)
	os.Stderr.Sync()

	return exitSuccess
}

func p(w io.Writer, format string, a ...interface{}) {
	fmt.Fprintf(w, format, a...)
}

// builtinCTypes mappings built-in C type to Go type.
var builtinCTypes = map[string]string{
	"size_t":             "uint64",         // size_t -> uint64
	"char":               "int8",           // char -> int8
	"signed char":        "int8",           // signed char -> int8
	"unsigned char":      "byte",           // unsigned char -> byte
	"short":              "int16",          // short -> int16
	"unsigned short":     "uint16",         // unsigned short -> uint16
	"long":               "int64",          // long -> int64
	"unsigned long":      "int64",          // unsigned long -> int64
	"signed long":        "uint",           // signed long -> uint
	"long long":          "int64",          // long long -> int64
	"unsigned long long": "uint64",         // unsigned long long -> uint64
	"signed long long":   "int64",          // signed long long -> int64
	"int":                "int32",          // int -> int32
	"unsigned int":       "uint32",         // unsigned int -> uint32
	"uint8_t":            "uint8",          // uint8_t -> uint8
	"uint16_t":           "uint16",         // uint16_t -> uint16
	"uint32_t":           "uint32",         // uint32_t -> uint32
	"uint64_t":           "uint64",         // uint64_t -> uint64
	"int8_t":             "int8",           // int8_t -> int8
	"int16_t":            "int16",          // int16_t -> int16
	"int32_t":            "int32",          // int32_t -> int32
	"int64_t":            "int64",          // int64_t -> int64
	"signed int":         "int32",          // signed int -> int32
	"short int":          "int16",          // short int -> int16
	"unsigned short int": "uint16",         // unsigned short int -> uint16
	"signed short int":   "int16",          // signed short int -> int16
	"long int":           "int64",          // long int -> int64
	"unsigned long int":  "uint64",         // unsigned long int -> uint64
	"signed long int":    "int64",          // signed long int -> int64
	"float":              "float32",        // float -> float32
	"double":             "float64",        // double -> float64
	"complex float":      "complex64",      // complex float -> complex64
	"complex double":     "complex128",     // complex double -> complex128
	"void*":              "*byte",          // void* -> *byte
	"void":               "unsafe.Pointer", // void -> unsafe.Pointer
	"_Bool":              "bool",           // _Bool -> bool
}

func convertGoType(s string) string {
	argType := strings.TrimPrefix(s, "const ")       // trim const
	argType = strings.TrimPrefix(argType, "struct ") // trim struct
	argType = strings.TrimPrefix(argType, "enum ")   // trim enum

	// special caes
	if !strings.Contains(argType, "void*") {
		argType = strings.TrimSuffix(argType, " *") // trim pointer
	}

	// fmt.Fprintf(os.Stderr, "%s: %s\n", argType, s)
	if goType, ok := builtinCTypes[argType]; ok {
		return goType
	}

	return export(argType) // export type
}

func export(s string) string {
	return string(strings.ToUpper(string(s[0])) + s[1:])
}

func upperCamelCase(s string) string {
	s = goCamelCase(strings.ToLower(s))

	first := s[0]
	if isASCIIDigit(first) {
		first = 'X' + '_' + first
	}

	return string(strings.ToUpper(string(first)) + s[1:])
}

func lowerCamelCase(s string) string {
	s = goCamelCase(s)

	first := s[0]
	if isASCIIDigit(first) {
		first = 'X' + '_' + first
	}

	return string(strings.ToLower(string(first)) + s[1:])
}

// goCamelCase camel-cases Go identifier name.
func goCamelCase(s string) string {
	var b []byte

	for i := 0; i < len(s); i++ {
		c := s[i]

		switch {
		case c == '.' && i+1 < len(s) && isASCIILower(s[i+1]):
			// skip over '.' in ".{{lowercase}}".

		case c == '.':
			b = append(b, '_') // convert '.' to '_'

		case c == '_' && (i == 0 || s[i-1] == '.'):
			// skip over '_'

		case c == '_' && i+1 < len(s) && isASCIILower(s[i+1]):
			// skip over '_' in "_{{lowercase}}".

		case isASCIIDigit(c):
			b = append(b, c)

		default:
			// Assume we have a letter now - if not, it's a bogus identifier.
			// The next word is a sequence of characters that must start upper case.
			if isASCIILower(c) {
				c -= 'a' - 'A' // convert lowercase to uppercase
			}
			b = append(b, c)

			// Accept lower case sequence that follows.
			for ; i+1 < len(s) && isASCIILower(s[i+1]); i++ {
				b = append(b, s[i+1])
			}
		}
	}

	return string(b)
}

func isASCIILower(c byte) bool { return 'a' <= c && c <= 'z' }

func isASCIIUpper(c byte) bool { return 'A' <= c && c <= 'Z' }

func isASCIIDigit(c byte) bool { return '0' <= c && c <= '9' }
