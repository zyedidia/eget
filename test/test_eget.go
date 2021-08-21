package main

import (
	"fmt"
	"os"
	"os/exec"
)

func fileExists(path string) error {
	_, err := os.Stat(path)
	return err
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	eget := os.Getenv("EGET_BIN")

	must(run(eget, "--system", "linux/amd64", "jgm/pandoc"))
	must(fileExists("pandoc"))

	must(run(eget, "zyedidia/micro", "--tag", "nightly", "--asset", "osx"))
	must(fileExists("micro"))

	must(run(eget, "-x", "--asset", "nvim.appimage", "--to", "nvim", "neovim/neovim"))
	must(fileExists("nvim"))

	must(run(eget, "--system", "darwin/amd64", "sharkdp/fd"))
	must(fileExists("fd"))

	must(run(eget, "--system", "windows/amd64", "--asset", "windows-gnu", "BurntSushi/ripgrep"))
	must(fileExists("rg.exe"))

	must(run(eget, "-f", "eget.1", "zyedidia/eget"))
	must(fileExists("eget.1"))

	fmt.Println("ALL TESTS PASS")
}
