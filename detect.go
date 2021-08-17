package main

import (
	"fmt"
	"regexp"
)

// A Detector selects an asset from a list of possibilities.
type Detector interface {
	Detect(assets []string) (string, []string, error)
}

type OS struct {
	name  string
	regex *regexp.Regexp
}

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

type Arch struct {
	name  string
	regex *regexp.Regexp
}

func (a *Arch) Match(s string) bool {
	return a.regex.MatchString(s)
}

var (
	ArchAMD64 = Arch{
		name:  "amd64",
		regex: regexp.MustCompile(`(?i)(x64|amd64|x86(-|_)64)`),
	}
	ArchI386 = Arch{
		name:  "i386",
		regex: regexp.MustCompile(`(?i)(x32|amd32|x86(-|_)32|i?386)`),
	}
	ArchArm = Arch{
		name:  "arm",
		regex: regexp.MustCompile(`(?i)(arm)`),
	}
	ArchArm64 = Arch{
		name:  "arm64",
		regex: regexp.MustCompile(`(?i)(arm64)`),
	}
	ArchRiscv64 = Arch{
		name:  "riscv64",
		regex: regexp.MustCompile(`(?i)(riscv64)`),
	}
)

var goarchmap = map[string]Arch{
	"amd64":   ArchAMD64,
	"386":     ArchI386,
	"arm":     ArchArm,
	"arm64":   ArchArm64,
	"riscv64": ArchRiscv64,
}

type AllDetector struct{}

func (a *AllDetector) Detect(assets []string) (string, []string, error) {
	all := make([]string, 0, len(assets))
	for _, asset := range assets {
		all = append(all, asset)
	}
	return "", all, fmt.Errorf("%d matches found", len(all))
}

type SystemDetector struct {
	Os   OS
	Arch Arch
}

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
