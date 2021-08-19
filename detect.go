package main

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// A Detector selects an asset from a list of possibilities.
type Detector interface {
	// Detect takes a list of possible assets and returns a direct match. If a
	// single direct match is not found, it returns a list of candidates and an
	// error explaining what happened.
	Detect(assets []string) (string, []string, error)
}

// An OS represents a target operating system.
type OS struct {
	name  string
	regex *regexp.Regexp
}

// Match returns true if the given archive name is likely to store a binary for
// this OS.
func (os *OS) Match(s string) bool {
	return os.regex.MatchString(s)
}

var (
	OSDarwin = OS{
		name:  "darwin",
		regex: regexp.MustCompile(`(?i)(darwin|macos|osx)`),
	}
	OSWindows = OS{
		name:  "windows",
		regex: regexp.MustCompile(`(?i)(win|windows)`),
	}
	OSLinux = OS{
		name:  "linux",
		regex: regexp.MustCompile(`(?i)(linux)`),
	}
	OSNetBSD = OS{
		name:  "netbsd",
		regex: regexp.MustCompile(`(?i)(netbsd)`),
	}
	OSFreeBSD = OS{
		name:  "freebsd",
		regex: regexp.MustCompile(`(?i)(freebsd)`),
	}
	OSOpenBSD = OS{
		name:  "openbsd",
		regex: regexp.MustCompile(`(?i)(openbsd)`),
	}
	OSAndroid = OS{
		name:  "android",
		regex: regexp.MustCompile(`(?i)(android)`),
	}
	OSIllumos = OS{
		name:  "illumos",
		regex: regexp.MustCompile(`(?i)(illumos)`),
	}
	OSSolaris = OS{
		name:  "solaris",
		regex: regexp.MustCompile(`(?i)(solaris)`),
	}
	OSPlan9 = OS{
		name:  "plan9",
		regex: regexp.MustCompile(`(?i)(plan9)`),
	}
)

// a map of GOOS values to internal OS matchers
var goosmap = map[string]OS{
	"darwin":  OSDarwin,
	"windows": OSWindows,
	"linux":   OSLinux,
	"netbsd":  OSNetBSD,
	"openbsd": OSOpenBSD,
	"freebsd": OSFreeBSD,
	"android": OSAndroid,
	"illumos": OSIllumos,
	"solaris": OSSolaris,
	"plan9":   OSPlan9,
}

// An Arch represents a system architecture, such as amd64, i386, arm or others.
type Arch struct {
	name  string
	regex *regexp.Regexp
}

// Match returns true if this architecture is likely supported by the given
// archive name.
func (a *Arch) Match(s string) bool {
	return a.regex.MatchString(s)
}

var (
	ArchAMD64 = Arch{
		name:  "amd64",
		regex: regexp.MustCompile(`(?i)(x64|amd64|x86(-|_)64)`),
	}
	ArchI386 = Arch{
		name:  "386",
		regex: regexp.MustCompile(`(?i)(x32|amd32|x86(-|_)32|i?386)`),
	}
	ArchArm = Arch{
		name:  "arm",
		regex: regexp.MustCompile(`(?i)(arm)`),
	}
	ArchArm64 = Arch{
		name:  "arm64",
		regex: regexp.MustCompile(`(?i)(arm64|armv8)`),
	}
	ArchRiscv64 = Arch{
		name:  "riscv64",
		regex: regexp.MustCompile(`(?i)(riscv64)`),
	}
)

// a map from GOARCH values to internal architecture matchers
var goarchmap = map[string]Arch{
	"amd64":   ArchAMD64,
	"386":     ArchI386,
	"arm":     ArchArm,
	"arm64":   ArchArm64,
	"riscv64": ArchRiscv64,
}

// AllDetector matches every asset. If there is only one asset, it is returned
// as a direct match. If there are multiple assets they are all returned as
// candidates.
type AllDetector struct{}

func (a *AllDetector) Detect(assets []string) (string, []string, error) {
	all := make([]string, 0, len(assets))
	for _, asset := range assets {
		all = append(all, asset)
	}
	if len(all) == 1 {
		return all[0], nil, nil
	}
	return "", all, fmt.Errorf("%d matches found", len(all))
}

// SingleAssetDetector finds a single named asset.
type SingleAssetDetector struct {
	Asset string
}

func (s *SingleAssetDetector) Detect(assets []string) (string, []string, error) {
	var candidates []string
	for _, a := range assets {
		if path.Base(a) == s.Asset {
			return a, nil, nil
		}
		if strings.Contains(path.Base(a), s.Asset) {
			candidates = append(candidates, a)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil, nil
	} else if len(candidates) > 1 {
		return "", candidates, fmt.Errorf("%d candidates found for asset `%s`", len(candidates), s.Asset)
	}
	return "", nil, fmt.Errorf("asset `%s` not found", s.Asset)
}

// A SystemDetector matches a particular OS/Arch system pair.
type SystemDetector struct {
	Os   OS
	Arch Arch
}

// NewSystemDetector returns a new detector for the given OS/Arch as given by
// Go OS/Arch names.
func NewSystemDetector(sos, sarch string) (*SystemDetector, error) {
	os, ok := goosmap[sos]
	if !ok {
		return nil, fmt.Errorf("unsupported target OS: %s", sos)
	}
	arch, ok := goarchmap[sarch]
	if !ok {
		return nil, fmt.Errorf("unsupported target arch: %s", sarch)
	}
	return &SystemDetector{
		Os:   os,
		Arch: arch,
	}, nil
}

// Detect extracts the assets that match this detector's OS/Arch pair. If one
// direct OS/Arch match is found, it is returned.  If multiple OS/Arch matches
// are found they are returned as candidates. If multiple assets that only
// match the OS are found, and no full OS/Arch matches are found, the OS
// matches are returned as candidates. Otherwise all assets are returned as
// candidates.
func (d *SystemDetector) Detect(assets []string) (string, []string, error) {
	var matches []string
	var candidates []string
	all := make([]string, 0, len(assets))
	for _, a := range assets {
		os := d.Os.Match(a)
		arch := d.Arch.Match(a)
		if os && arch {
			matches = append(matches, a)
		}
		if os {
			candidates = append(candidates, a)
		}
		all = append(all, a)
	}
	if len(matches) == 1 {
		return matches[0], nil, nil
	} else if len(matches) > 1 {
		return "", matches, fmt.Errorf("%d matches found", len(matches))
	} else if len(candidates) > 0 {
		return "", candidates, fmt.Errorf("%d candidates found (unsure architecture)", len(candidates))
	}
	return "", all, fmt.Errorf("no candidates found")
}
