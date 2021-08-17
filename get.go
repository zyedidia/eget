package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	pb "github.com/schollz/progressbar/v3"
)

func fatal(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}

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

	var output io.Writer = os.Stdout
	if opts.Quiet {
		output = io.Discard
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
			fatal("invalid repo (no '/' found)")
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
		fatal(err)
	}

	// Determine the appropriate detector. If the --system is 'all', we use an
	// AllDetector, which will just return all assets. Otherwise we use the
	// --system pair provided by the user, or the runtime.GOOS/runtime.GOARCH
	// pair by default (the host system OS/Arch pair).
	var detector Detector
	if opts.Asset != "" {
		detector = &SingleAssetDetector{
			Asset: opts.Asset,
		}
	} else if opts.System == "all" {
		detector = &AllDetector{}
	} else if opts.System != "" {
		split := strings.Split(opts.System, "/")
		if len(split) < 2 {
			fatal("system descriptor must be os/arch")
		}
		detector, err = NewSystemDetector(split[0], split[1])
	} else {
		detector, err = NewSystemDetector(runtime.GOOS, runtime.GOARCH)
	}
	if err != nil {
		fatal(err)
	}

	// get the url and candidates from the detector
	url, candidates, err := detector.Detect(assets)
	if len(candidates) != 0 && err != nil {
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
	} else if err != nil {
		fatal(err)
	}

	// print the URL
	fmt.Fprintf(output, "%s\n", url)

	// download with progress bar
	buf := &bytes.Buffer{}
	err = Download(url, buf, func(size int64) *pb.ProgressBar {
		var pbout io.Writer = os.Stderr
		if opts.Quiet {
			pbout = io.Discard
		}
		return pb.NewOptions64(size,
			pb.OptionSetWriter(pbout),
			pb.OptionShowBytes(true),
			pb.OptionSetWidth(10),
			pb.OptionThrottle(65*time.Millisecond),
			pb.OptionShowCount(),
			pb.OptionSpinnerType(14),
			pb.OptionFullWidth(),
			pb.OptionSetDescription("Downloading"),
			pb.OptionOnCompletion(func() {
				fmt.Fprint(pbout, "\n")
			}),
			pb.OptionSetTheme(pb.Theme{
				Saucer:        "=",
				SaucerHead:    ">",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))
	})
	if err != nil {
		fatal(fmt.Sprintf("%s (URL: %s)", err, url))
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
	if len(bins) != 0 && err != nil {
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
	} else if err != nil {
		fatal(err)
	}

	// write the extracted file to a file on disk, in the --to directory if
	// requested
	var out string
	if opts.Rename != "" {
		out = opts.Rename
	} else {
		out = filepath.Base(bin.Name)
	}
	if !filepath.IsAbs(out) && opts.Output != "" {
		out = filepath.Join(opts.Output, out)
	}

	if opts.Exec {
		bin.Mode |= 0111
	}

	// write the file using the same perms it had in the archive
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, bin.Mode)
	if err != nil {
		fatal(err)
	}
	f.Write(bin.Data)
	f.Close()

	fmt.Fprintf(output, "Extracted `%s` to `%s`\n", bin.Name, out)
}
