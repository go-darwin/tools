// Copyright 2021 The Go Darwin Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package xcoderelease provides the list of Xcode releases from the xcodereleases.com.
package xcoderelease

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	json "github.com/goccy/go-json"
)

var dataURI = &url.URL{
	Scheme: "https",
	Host:   "xcodereleases.com",
	Path:   "data.json",
}

// DownloadJSON downoads xcodereleases data.json.
func DownloadJSON(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dataURI.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Unmarshal parses the xcodereleases.com JSON-encoded data and returns the new Xcoderelease.
func Unmarshal(data []byte) (xrs []*XcodeRelease, err error) {
	if err := json.UnmarshalNoEscape(data, &xrs); err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}

	return xrs, nil
}

// XcodeRelease represents a Xcode release information.
type XcodeRelease struct {
	Name      string     `json:"name"`
	SDKs      *SDKs      `json:"sdks"`
	Version   Version    `json:"version"`
	Requires  string     `json:"requires"`
	Compilers *Compilers `json:"compilers"`
	Checksums Checksum   `json:"checksums"`
	Date      Date       `json:"date"`
	Links     Link       `json:"links"`
}

// SDKs represents a Apple each OS sdks.
type SDKs struct {
	MacOS   []MacOS   `json:"macOS"`
	IOS     []IOS     `json:"iOS"`
	TvOS    []TvOS    `json:"tvOS"`
	WatchOS []WatchOS `json:"watchOS"`
}

// MacOS represents a details of macOS sdk.
type MacOS struct {
	Build   string `json:"build"`
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// IOS represents a details of iOS sdk.
type IOS struct {
	Build   string `json:"build"`
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// TvOS represents a details of TvOS sdk.
type TvOS struct {
	Build   string `json:"build"`
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// WatchOS represents a details of WatchOS sdk.
type WatchOS struct {
	Build   string `json:"build"`
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// Version represents a release version.
type Version struct {
	Build   string   `json:"build"`
	Number  string   `json:"number"`
	Release *Release `json:"release,omitempty"`
}

// Release represents a release of version.
type Release struct {
	Beta    int  `json:"beta,omitempty"`
	Dp      int  `json:"dp,omitempty"`
	Gm      bool `json:"gm"`
	GmSeed  int  `json:"gmSeed,omitempty"`
	Rc      int  `json:"rc,omitempty"`
	Release bool `json:"release"`
}

// Compilers represents a compilers information of Xcode.
type Compilers struct {
	Clang   []ClangCompiler   `json:"clang"`
	Gcc     []GCCCompiler     `json:"gcc,omitempty"`
	Llvm    []LLVMCompiler    `json:"llvm,omitempty"`
	LlvmGcc []LLVMGCCCompiler `json:"llvm_gcc,omitempty"`
	Swift   []SwiftCompiler   `json:"swift"`
}

// ClangCompiler represents a clang compiler information of Xcode.
type ClangCompiler struct {
	Build   string `json:"build"`
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// ClangCompiler represents a GCC compiler information of Xcode.
type GCCCompiler struct {
	Build   string `json:"build"`
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// ClangCompiler represents a LLVM compiler information of Xcode.
type LLVMCompiler struct {
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// LLVMGCCCompiler represents a LLVM GCC compiler information of Xcode.
type LLVMGCCCompiler struct {
	Build   string `json:"build"`
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// SwiftCompiler represents a swift compiler information of Xcode.
type SwiftCompiler struct {
	Build   string `json:"build"`
	Number  string `json:"number"`
	Release bool   `json:"release"`
}

// Checksum checksum of Xcode xip tarball.
type Checksum struct {
	Sha1 string `json:"sha1,omitempty"`
}

// Date represents a release date of Xcode release.
type Date struct {
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
}

// Link represents a link of Xcode release.
type Link struct {
	Download URL `json:"download"`
	Notes    URL `json:"notes"`
}

// URL represents a url of Xcode release.
type URL struct {
	URL string `json:"url"`
}
