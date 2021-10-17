// Copyright 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build ignore

// ctypes for godef source for mapping the common C type to Go type.
package ctypes

/*
#include <stdint.h>
#include <stdbool.h>
#include <complex.h>

typedef signed char signedChar;
typedef unsigned char unsignedChar;
typedef unsigned short unsignedShort;
typedef unsigned long unsignedLong;
typedef signed long signedLong;
typedef long long longLong;
typedef unsigned long long unsignedLongLong;
typedef signed long long signedLongLong;
typedef unsigned int unsignedInt;
typedef signed int signedInt;
typedef short int shortInt;
typedef unsigned short int unsignedShortInt;
typedef signed short int signedShortInt;
typedef long int longInt;
typedef unsigned long int unsignedLongInt;
typedef signed long int signedLongInt;
typedef complex float complexFloat;
typedef complex double complexDouble;
typedef long complex double longComplexDouble;
typedef void * voidptr;
*/
import "C"

type Size_t C.size_t
type Char C.char
type signedChar C.signedChar
type UnsignedChar C.unsignedChar
type short C.short
type UnsignedShort C.unsignedShort
type Long C.long
type UnsignedLong C.unsignedLong
type SignedLong C.signedLong
type LongLong C.longLong
type UnsignedLongLong C.unsignedLongLong
type SignedLongLong C.signedLongLong
type Int C.int
type UnsignedInt C.unsignedInt
type Uint8 C.uint8_t
type Uint16 C.uint16_t
type Uint32 C.uint32_t
type Uint64 C.uint64_t
type Int8 C.int8_t
type Int16 C.int16_t
type Int32 C.int32_t
type Int64 C.int64_t
type SignedInt C.signedInt
type ShortInt C.shortInt
type UnsignedShortInt C.unsignedShortInt
type SignedShortInt C.signedShortInt
type LongInt C.longInt
type UnsignedLongInt C.unsignedLongInt
type SignedLongInt C.signedLongInt
type Float C.float
type Double C.double
type ComplexFloat C.complexFloat
type ComplexDouble C.complexDouble
type VoidPtr C.voidptr
type Void C.void
type Bool C._Bool
