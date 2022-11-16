package home

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"
)

func Home() (string, error) {
	userData, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("find homedir: %w", err)
	}
	return userData.HomeDir, err
}

// Expand takes a path as input and replaces ~ at the start of the path with the user's
// home directory. Does nothing if the path does not start with '~'.
func Expand(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	var userData *user.User
	var err error

	homeString := strings.Split(filepath.ToSlash(path), "/")[0]
	if homeString == "~" {
		userData, err = user.Current()
		if err != nil {
			return "", fmt.Errorf("expand tilde: %w", err)
		}
	} else {
		userData, err = user.Lookup(homeString[1:])
		if err != nil {
			return "", fmt.Errorf("expand tilde: %w", err)
		}
	}

	home := userData.HomeDir

	return strings.Replace(path, homeString, home, 1), nil
}
