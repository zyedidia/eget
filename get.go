package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var tagf = flag.String("tag", "", "tagged release to use")
var output = flag.String("o", "", "output file")
var yes = flag.Bool("y", false, "automatically approve all yes/no prompts")

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
	url, candidates, err := detector.Detect(assets)

	if err != nil {
		fmt.Printf("%v: please select manually\n", err)
		for i, c := range candidates {
			fmt.Printf("(%d) %s\n", i+1, path.Base(c))
		}
		var choice int
		for {
			fmt.Print("Please select one (enter its number): ")
			_, err := fmt.Scanf("%d", &choice)
			if err == nil && (choice <= 0 || choice > len(candidates)) {
				err = fmt.Errorf("%d is out of bounds", choice)
			}
			if err == nil {
				break
			}
			fmt.Printf("Invalid selection: %v\n", err)
		}
		url = candidates[choice-1]
	}

	fmt.Printf("%s\n", url)
	fmt.Print("Download and continue? [Y/n]")

	if !*yes {
		var input string
		fmt.Scanln(&input)
		input = strings.ToLower(strings.TrimSpace(input))
		if input != "" && !strings.HasPrefix(input, "y") && !strings.HasPrefix(input, "yes") {
			fmt.Println("Operation canceled")
			os.Exit(0)
		}
	}

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

	extractor := NewExtractor(path.Base(url), &BinaryChooser{})
	if err != nil {
		log.Fatal(err)
	}
	bin, err := extractor.Extract(body)
	if err != nil {
		log.Fatal(err)
	}

	var out string
	if output != nil && *output != "" {
		out = *output
	} else {
		out = filepath.Base(bin.Name)
	}

	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, bin.Mode)
	if err != nil {
		log.Fatal(err)
	}
	f.Write(bin.Data)
	f.Close()

	fmt.Printf("Extracted `%s` to `%s`\n", bin.Name, out)
}
