package main

import (
	"fmt"
	"regexp"
	"runtime"
)

// A Detector selects an asset from a list of possibilities.
type Detector interface {
	Detect(assets []string) (string, error)
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
		regex: regexp.MustCompile(`(darwin|macos|osx)`),
	}
	OSWindows = OS{
		name:  "windows",
		regex: regexp.MustCompile(`(win|windows)`),
	}
	OSLinux = OS{
		name:  "linux",
		regex: regexp.MustCompile(`(linux)`),
	}
)

var goosmap = map[string]OS{
	"darwin":  OSDarwin,
	"windows": OSWindows,
	"linux":   OSLinux,
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
		regex: regexp.MustCompile(`(x64|amd64|x86(-|_)64)`),
	}
	ArchI386 = Arch{
		name:  "i386",
		regex: regexp.MustCompile(`(x32|amd32|x86(-|_)32|i?386)`),
	}
)

var goarchmap = map[string]Arch{
	"amd64": ArchAMD64,
	"i386":  ArchI386,
}

type SystemDetector struct {
	Os   OS
	Arch Arch
}

func NewHostDetector() (*SystemDetector, error) {
	os, ok := goosmap[runtime.GOOS]
	if !ok {
		return nil, fmt.Errorf("unsupported host OS: %s", runtime.GOOS)
	}
	arch, ok := goarchmap[runtime.GOARCH]
	if !ok {
		return nil, fmt.Errorf("unsupported host arch: %s", runtime.GOARCH)
	}
	return &SystemDetector{
		Os:   os,
		Arch: arch,
	}, nil
}

type MultipleCandidatesError struct {
	Candidates []string
}

func (me *MultipleCandidatesError) Error() string {
	return fmt.Sprintf("found %d candidates", len(me.Candidates))
}

func (d *SystemDetector) Detect(assets []string) (string, error) {
	var matches []string
	for _, a := range assets {
		if d.Os.Match(a) && d.Arch.Match(a) {
			matches = append(matches, a)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return "", &MultipleCandidatesError{
			Candidates: matches,
		}
	}
	return "", fmt.Errorf("no candidates found")
}
