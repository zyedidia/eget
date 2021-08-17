package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jessevdk/go-flags"
	pb "github.com/schollz/progressbar/v3"
)

func main() {
	flagparser := flags.NewParser(&opts, flags.PassDoubleDash|flags.PrintErrors)
	flagparser.Usage = "[OPTIONS] REPO"
	args, err := flagparser.Parse()
	if err != nil {
		os.Exit(1)
	}

	if opts.Version {
		fmt.Println("get version", Version)
		os.Exit(0)
	}

	if opts.Help {
		flagparser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	if len(args) <= 0 {
		fmt.Println("no get target given")
		flagparser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	repo := args[0]
	if !strings.Contains(repo, "/") {
		log.Fatal("invalid repo (no '/' found)")
	}
	repoparts := strings.Split(repo, "/")

	tag := "latest"
	if opts.Tag != "" {
		tag = fmt.Sprintf("tags/%s", opts.Tag)
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
	if opts.System == "all" {
		detector = &AllDetector{}
	} else if opts.System != "" {
		split := strings.Split(opts.System, "/")
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
			fmt.Print("Enter selection number: ")
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

	if !opts.Yes {
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
	if opts.ExtractFile != "" {
		extractor = NewExtractor(path.Base(url), &LiteralFileChooser{
			File: opts.ExtractFile,
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
	if opts.Output != "" {
		out = opts.Output
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
