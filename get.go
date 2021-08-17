package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
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

	// Determine the appropriate Finder to use. If opts.URL is provided, we use
	// a DirectAssetFinder. Otherwise we use a GithubAssetFinder. When a Github
	// repo is provided, we assume the repo name is the 'tool' name (for direct
	// URLs, the tool name is unknown and remains empty).
	var finder Finder
	var tool string
	if opts.URL {
		finder = &DirectAssetFinder{
			URL: args[0],
		}
		opts.System = "all"
	} else {
		repo := args[0]
		if !strings.Contains(repo, "/") {
			log.Fatal("invalid repo (no '/' found)")
		}
		tool = strings.Split(repo, "/")[1]

		tag := "latest"
		if opts.Tag != "" {
			tag = fmt.Sprintf("tags/%s", opts.Tag)
		}

		finder = &GithubAssetFinder{
			Repo: repo,
			Tag:  tag,
		}
	}
	assets, err := finder.Find()
	if err != nil {
		log.Fatal(err)
	}

	// Determine the appropriate detector. If the --system is 'all', we use an
	// AllDetector, which will just return all assets. Otherwise we use the
	// --system pair provided by the user, or the runtime.GOOS/runtime.GOARCH
	// pair by default (the host system OS/Arch pair).
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

	// get the url and candidates from the detector
	url, candidates, err := detector.Detect(assets)
	if err != nil {
		// if multiple candidates are returned, the user must select manually which one to download
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

	// print the URL and ask for confirmation to continue before downloading
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

	// download with progress bar
	buf := &bytes.Buffer{}
	err = Download(url, buf, func(size int64) *pb.ProgressBar {
		return pb.DefaultBytes(size, "Downloading")
	})
	if err != nil {
		log.Fatalf("%s (URL: %s)\n", err, url)
	}

	body := buf.Bytes()

	if opts.Hash {
		sum := sha256.Sum256(body)
		fmt.Printf("%x\n", sum)
	}

	// Determine which extractor to use. If --download-only is provided, we
	// just "extract" the downloaded archive to itself. Otherwise we try to
	// extract the literal file provided by --file, or by default we just
	// extract a binary with the tool name that was possibly auto-detected
	// above.
	var extractor Extractor
	if opts.DLOnly {
		extractor = &SingleFileExtractor{
			Name: path.Base(url),
			Decompress: func(r io.Reader) (io.Reader, error) {
				return r, nil
			},
		}
	} else if opts.ExtractFile != "" {
		extractor = NewExtractor(path.Base(url), &LiteralFileChooser{
			File: opts.ExtractFile,
		})
	} else {
		extractor = NewExtractor(path.Base(url), &BinaryChooser{
			Tool: tool,
		})
	}

	// extract the binary information
	bin, bins, err := extractor.Extract(body)
	if err != nil {
		// if there are multiple candidates, have the user select manually
		fmt.Printf("%v: please select manually\n", err)
		for i, c := range bins {
			fmt.Printf("(%d) %s\n", i+1, c.Name)
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
		bin = bins[choice-1]
	}

	// write the extracted file to a file on disk, in the --to directory if
	// requested
	out := filepath.Base(bin.Name)
	if opts.Output != "" {
		out = filepath.Join(opts.Output, out)
	}

	// write the file using the same perms it had in the archive
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, bin.Mode)
	if err != nil {
		log.Fatal(err)
	}
	f.Write(bin.Data)
	f.Close()

	fmt.Printf("Extracted `%s` to `%s`\n", bin.Name, out)
}
