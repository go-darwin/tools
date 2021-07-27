// SPDX-FileCopyrightText: 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build darwin && amd64 && gc
// +build darwin,amd64,gc

// Package godefs generates "go tool cgo -godefs" template file from the parses C include headers built on top of libclang.
package godefs

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-clang/clang-v3.9/clang"
)

// If Debug is true, debugging output is printed to stderr.
var Debug = false // default: false

func init() {
	spew.Config = spew.ConfigState{
		Indent:                  "  ",
		SortKeys:                true, // maps should be spewed in a deterministic order
		DisableMethods:          false,
		DisablePointerMethods:   false,
		DisablePointerAddresses: false, // don't spew the addresses of pointers
		DisableCapacities:       false, // don't spew capacities of collections
		ContinueOnMethod:        true,  // recursion should continue once a custom error or Stringer interface is invoked
		SpewKeys:                false, // if unable to sort map keys then spew keys to strings and sort those
		MaxDepth:                4,     // maximum number of levels to descend into nested data structures.
	}
}

const ptrsize = 8 // Pointer size. All supported platforms are 64-bit.

type cparser struct {
	out  *bytes.Buffer
	bufs map[token.Token]*bytes.Buffer

	skipNextKind bool // for skip next kind
	universes    map[string]bool
	seemName     map[string]bool
}

const packageTmpl = `
// SPDX-FileCopyrightText: 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build ignore
// +build ignore

// Input to cgo -godefs.

package %s

import "C"

`

const unsavedFileFmt = `
#include <%s>
`

// MakeGodefs makes godefs template Go source from C inclued header.
func MakeGodefs(packageName string, headerfiles []string) error {
	c := &cparser{
		out: bytes.NewBufferString(fmt.Sprintf(packageTmpl, packageName)),
		bufs: map[token.Token]*bytes.Buffer{
			token.CONST: new(bytes.Buffer),
			token.FUNC:  new(bytes.Buffer),
		},
		universes: make(map[string]bool),
		seemName:  make(map[string]bool),
	}

	// store all predeclared object names
	for _, tok := range []token.Token{token.PACKAGE, token.IMPORT, token.CONST, token.FUNC, token.VAR, token.TYPE} {
		c.universes[tok.String()] = true
	}
	for _, s := range types.Universe.Names() {
		c.universes[s] = true
	}

	idx := clang.NewIndex(1, 0)
	defer idx.Dispose()

	for _, headerfile := range headerfiles {
		fname, err := filepath.Abs(headerfile)
		if err != nil {
			return fmt.Errorf("parse %s abs path: %w", headerfile, err)
		}

		unsaved := fmt.Sprintf(unsavedFileFmt, filepath.Base(fname))

		if err := c.ParseCFile(idx, headerfile, "unsaved.c",
			WithCommandLineArgs([]string{fmt.Sprintf("-I%s", filepath.Dir(fname))}),
			WithUnsavedFiles([]clang.UnsavedFile{clang.NewUnsavedFile("unsaved.c", string(unsaved))}),
		); err != nil {
			return fmt.Errorf("parse %s C file: %w", fname, err)
		}
	}

	buf := c.out.Bytes()
	data, err := format.Source(buf)
	if err != nil {
		os.Stdout.Write(buf) // for invastigate
		return fmt.Errorf("format source: %w", err)
	}
	os.Stdout.Write(data)

	// in = string(in1) + string(in2)
	// if err := writeASMFile(in, fmt.Sprintf("zsyscall_darwin_%s.s", arch), "go1.13"); err != nil {
	// 	return fmt.Errorf("failed to writeASMFile: %w", err)
	// }

	return nil
}

// StripDefine strips unnecessary C define macro to go tool cgo -godefs input source.
func (c *cparser) StripDefine(src []byte) []byte {
	return nil
}

// DefaultTranslationUnitFlags default of libclang translation unit flag.
const DefaultTranslationUnitFlags = clang.TranslationUnit_Flags(
	clang.TranslationUnit_DetailedPreprocessingRecord |
		clang.TranslationUnit_Incomplete |
		clang.TranslationUnit_PrecompiledPreamble |
		clang.TranslationUnit_ForSerialization |
		clang.TranslationUnit_CXXChainedPCH |
		clang.TranslationUnit_SkipFunctionBodies |
		clang.TranslationUnit_CreatePreambleOnFirstParse |
		clang.TranslationUnit_KeepGoing,
)

// DefaultCommandLineArgs default of libclang translation unit command line args.
var DefaultCommandLineArgs = []string{
	// token from https://github.com/golang/go/blob/go1.16.6/src/syscall/types_darwin.go#L17-L19
	"-U__DARWIN_UNIX03", // undefine __DARWIN_UNIX03
	"-DKERNEL",
	"-D_DARWIN_USE_64_BIT_INODE",

	// token from https://github.com/golang/go/blob/go1.16.6/src/cmd/cgo/gcc.go#L1551-L1555
	"-arch", "x86_64",
	"-m64",

	// token from https://github.com/golang/go/blob/go1.16.6/src/cmd/cgo/gcc.go#L1584-L1589
	"-w",         // no warnings
	"-Wno-error", // warnings are not errors
	"-xc",        // input language is C

	// token from https://github.com/golang/go/blob/go1.16.6/src/cmd/cgo/gcc.go#L1593-L1607
	"-ferror-limit=0",
	"-Wno-unknown-warning-option",
	"-Wno-unneeded-internal-declaration",
	"-Wno-unused-function",
	"-Qunused-arguments",
	"-fno-builtin",

	// token from https://github.com/golang/go/blob/go1.17rc1/src/cmd/cgo/gcc.go#L1642
	"-fno-lto",
}

// option represents a optional values of ParseCFile.
type option struct {
	commandLineArgs []string
	unsavedFiles    []clang.UnsavedFile
	tuFlags         clang.TranslationUnit_Flags
}

// ParseOption an Option configures a ParseCFile.
type ParseOption interface {
	apply(o *option)
}

// optionFunc wraps a func so it satisfies the ParseOptios interface.
type optionFunc func(o *option)

// apply implements ParseOption.apply.
func (f optionFunc) apply(o *option) {
	f(o)
}

// WithUnsavedFiles appneds unsavedFiles to parse file.
func WithUnsavedFiles(unsavedFiles []clang.UnsavedFile) ParseOption {
	return optionFunc(func(o *option) {
		o.unsavedFiles = unsavedFiles
	})
}

// WithCommandLineArgs pass commandLineArgs to translation unit.
func WithCommandLineArgs(commandLineArgs []string) ParseOption {
	return optionFunc(func(o *option) {
		// C compile flags are last win
		o.commandLineArgs = append(o.commandLineArgs, commandLineArgs...)
	})
}

// ParseCFile parses C include header file recursively.
//
// This function can pass an unsavedFile such as a dummy in-memory C source.
func (c *cparser) ParseCFile(idx clang.Index, headerfile, filename string, opts ...ParseOption) error {
	o := &option{
		commandLineArgs: DefaultCommandLineArgs,
		tuFlags:         DefaultTranslationUnitFlags,
	}

	for _, opt := range opts {
		opt.apply(o)
	}

	tu := idx.ParseTranslationUnit(filename, o.commandLineArgs, o.unsavedFiles, uint32(o.tuFlags))
	defer tu.Dispose()

	cursor := tu.TranslationUnitCursor()

	cursor.Visit(clang.CursorVisitor(func(cursor, parent clang.Cursor) clang.ChildVisitResult {
		if cursor.IsNull() {
			return clang.ChildVisit_Continue
		}

		switch kind := cursor.Kind(); kind {
		case clang.Cursor_FunctionDecl:
			fmt.Fprintf(c.bufs[token.FUNC], "// %s %s(", cursor.ResultType().Spelling(), cursor.Spelling())

			numArgs := cursor.NumArguments()
			for i := int32(0); i < numArgs; i++ {
				arg := cursor.Argument(uint32(i))
				fmt.Fprintf(c.bufs[token.FUNC], "%s %s", arg.Type().Spelling(), arg.Spelling())
				if i+1 < numArgs {
					fmt.Fprintf(c.bufs[token.FUNC], ", ")
				}
			}
			fmt.Fprintf(c.bufs[token.FUNC], ")\n")

			return clang.ChildVisit_Recurse

		case clang.Cursor_MacroDefinition:
			// skip InclusionDirective macro
			if c.skipNextKind {
				defer func() { c.skipNextKind = false }()

				return clang.ChildVisit_Continue
			}

			// skip C built-in macro
			if file, _, _, _ := cursor.Location().FileLocation(); file.Name() == "" {
				return clang.ChildVisit_Continue
			}

			goSpelling := strings.TrimLeft(cursor.Spelling(), "_")
			goSpelling = ToExport(goSpelling)

			// guard Go built-in keywords
			if c.universes[goSpelling] {
				for {
					goSpelling = goSpelling + "_"
					if !c.universes[goSpelling] {
						break
					}
				}
			}

			// guard duplicate variable
			if c.seemName[goSpelling] {
				return clang.ChildVisit_Continue
			}
			c.seemName[goSpelling] = true

			fmt.Fprintf(c.bufs[token.CONST], "\t%s = C.%s\n", goSpelling, cursor.Spelling())

			return clang.ChildVisit_Continue

		case clang.Cursor_InclusionDirective:
			// skip next InclusionDirective macro
			c.skipNextKind = true
			return clang.ChildVisit_Continue

		default:
			if Debug {
				fmt.Printf("%s: %s\n", kind.Spelling(), cursor.Spelling())
			}
			return clang.ChildVisit_Continue
		}
	}))

	// write constants
	if c.bufs[token.CONST].Len() > 0 {
		c.out.WriteString(fmt.Sprintf("// from %s.\n", filepath.Base(headerfile)))
		c.out.WriteString("const (\n")
		c.out.ReadFrom(c.bufs[token.CONST])
		c.out.WriteString(")\n\n")
		c.bufs[token.CONST].Reset()
	}

	// write functions
	if c.bufs[token.FUNC].Len() > 0 {
		c.out.WriteString(fmt.Sprintf("// from %s.\n", filepath.Base(headerfile)))
		c.out.ReadFrom(c.bufs[token.FUNC])
		c.bufs[token.FUNC].Reset()
	}

	return nil
}

// ToExport replace s to Go export name.
//
// This function token from https://github.com/golang/go/blob/go1.16.6/src/cmd/cgo/gcc.go#L2970-L2979.
func ToExport(s string) string {
	if s == "" {
		return ""
	}

	r, size := utf8.DecodeRuneInString(s)
	if r == '_' {
		return "X" + s
	}

	return string(unicode.ToUpper(r)) + s[size:]
}

// GoCamelCase camel-cases a s name for use as a Go identifier.
//
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
//
// This function token from https://github.com/protocolbuffers/protobuf-go/blob/v1.27.1/internal/strs/strings.go.
func GoCamelCase(s string) string {
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]

		switch {
		case c == '.' && i+1 < len(s) && isASCIILower(s[i+1]):
			// Skip over '.' in ".{{lowercase}}".

		case c == '.':
			b = append(b, '_') // convert '.' to '_'

		case c == '_' && (i == 0 || s[i-1] == '.'):
			// Convert initial '_' to ensure we start with a capital letter.
			// Do the same for '_' after '.' to match historic behavior.
			b = append(b, 'X') // convert '_' to 'X'

		case c == '_' && i+1 < len(s) && isASCIILower(s[i+1]):
			// Skip over '_' in "_{{lowercase}}".

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
