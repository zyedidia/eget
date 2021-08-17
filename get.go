package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	pb "github.com/schollz/progressbar/v3"
)

var tagf = flag.String("tag", "", "tagged release to use")
var output = flag.String("o", "", "output file")
var yes = flag.Bool("y", false, "automatically approve all yes/no prompts")
var system = flag.String("system", "", "target system for the binary")
var exfile = flag.String("file", "", "file name to extract")

func main() {
	flag.Parse()

	if len(flag.Args()) <= 0 {
		fmt.Println("no get target given")
		os.Exit(0)
	}

	repo := flag.Args()[0]
	if !strings.Contains(repo, "/") {
		log.Fatal("invalid repo (no '/' found)")
	}
	repoparts := strings.Split(repo, "/")

	tag := "latest"
	if tagf != nil && *tagf != "" {
		tag = fmt.Sprintf("tags/%s", *tagf)
	}

	gh := &GithubAssetFinder{
		Repo: repo,
		Tag:  tag,
	}
	assets, err := gh.Find()
	if err != nil {
		log.Fatal(err)
	}

	var detector Detector
	if system != nil && *system == "all" {
		detector = &AllDetector{}
	} else if system != nil && *system != "" {
		split := strings.Split(*system, "/")
		if len(split) < 2 {
			log.Fatal("system descriptor must be os/arch")
		}
		detector, err = NewSystemDetector(split[0], split[1])
	} else {
		detector, err = NewSystemDetector(runtime.GOOS, runtime.GOARCH)
	}
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

	if !*yes {
		fmt.Print("Download and continue? [Y/n] ")

		var input string
		fmt.Scanln(&input)
		input = strings.ToLower(strings.TrimSpace(input))
		if input != "" && !strings.HasPrefix(input, "y") && !strings.HasPrefix(input, "yes") {
			fmt.Println("Operation canceled")
			os.Exit(0)
		}
	}

	buf := &bytes.Buffer{}
	err = Download(url, buf, func(size int64) *pb.ProgressBar {
		return pb.DefaultBytes(size, "Downloading")
	})
	if err != nil {
		log.Fatalf("%s (URL: %s)\n", err, url)
	}

	body := buf.Bytes()

	var extractor Extractor
	if exfile != nil && *exfile != "" {
		extractor = NewExtractor(path.Base(url), &LiteralFileChooser{
			File: *exfile,
		})
	} else {
		extractor = NewExtractor(path.Base(url), &BinaryChooser{
			Tool: repoparts[1],
		})
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
