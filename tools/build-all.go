//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

type Target struct {
	OS   string
	Arch string
}

func main() {
	targets := []struct {
		OS   string
		Arch string
	}{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"freebsd", "amd64"},
		{"linux", "amd64"},
		{"linux", "386"},
		{"linux", "arm64"},
		{"linux", "arm"},
		{"openbsd", "amd64"},
		{"windows", "amd64"},
		{"windows", "386"},
	}

	compile := func(platform, architecture string, wg *sync.WaitGroup) {
		defer wg.Done()

		cmd := exec.Command("make", "package")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		cgo := "0"
		if runtime.GOOS == "darwin" {
			cgo = "1"
		} else {
			fmt.Println("warning: it is recommended to cross-compile on Mac, for cgo")
		}
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("GOOS=%s", platform),
			fmt.Sprintf("GOARCH=%s", architecture),
			fmt.Sprintf("GOMAXPROCS=%d", runtime.NumCPU()),
			fmt.Sprintf("CGO_ENABLED=%s", cgo),
		)

		err := cmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		fmt.Printf("finished building %s-%s\n", platform, architecture)
	}

	var wg sync.WaitGroup

	wg.Add(len(targets))

	for _, t := range targets {
		fmt.Printf("starting build for %s-%s\n", t.OS, t.Arch)
		go compile(t.OS, t.Arch, &wg)
	}

	wg.Wait()
}
