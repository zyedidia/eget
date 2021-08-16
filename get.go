package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

var tagf = flag.String("tag", "", "tagged release to use")
var output = flag.String("o", "", "output file")

func main() {
	flag.Parse()

	if len(flag.Args()) <= 0 {
		fmt.Println("no get target given")
		os.Exit(0)
	}

	repo := flag.Args()[0]
	tag := "latest"
	if tagf != nil && *tagf != "" {
		tag = *tagf
	}

	gh := &GithubAssetFinder{
		Repo: repo,
		Tag:  tag,
	}
	assets, err := gh.Find()
	if err != nil {
		log.Fatal(err)
	}

	detector, err := NewHostDetector()
	if err != nil {
		log.Fatal(err)
	}
	url, err := detector.Detect(assets)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Downloading %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("server returned bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	br := bytes.NewReader(body)
	gzf, err := gzip.NewReader(br)
	if err != nil {
		log.Fatal(err)
	}
	extractor := &TarExtractor{
		File: &BinaryChooser{},
	}
	bin, err := extractor.Extract(gzf)
	if err != nil {
		log.Fatal(err)
	}

	out := bin.Name
	if output != nil && *output != "" {
		out = *output
	}

	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, bin.Mode)
	if err != nil {
		log.Fatal(err)
	}
	f.Write(bin.Data)
	f.Close()

	fmt.Printf("Extracted binary to %s\n", out)
}
