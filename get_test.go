package main

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {
	finder := &GithubAssetFinder{
		Repo: "zyedidia/micro",
		Tag:  "latest",
	}

	assets, err := finder.Find()
	if err != nil {
		t.Fatal(err)
	}

	detector, err := NewSystemDetector("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(detector.Detect(assets))
}
