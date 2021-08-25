package main

type Flags struct {
	Tag         string `short:"t" long:"tag" description:"tagged release to use instead of latest"`
	Prerelease  bool   `long:"pre-release" description:"include pre-releases when fetching the latest version"`
	Output      string `long:"to" description:"move to given location after extracting"`
	System      string `short:"s" long:"system" description:"target system to download for (use \"all\" for all choices)"`
	ExtractFile string `short:"f" long:"file" description:"file name to extract"`
	Quiet       bool   `short:"q" long:"quiet" description:"only print essential output"`
	DLOnly      bool   `long:"download-only" description:"stop after downloading the asset (no extraction)"`
	Asset       string `long:"asset" description:"download a specific asset containing the given string"`
	Hash        bool   `long:"sha256" description:"show the SHA-256 hash of the downloaded asset"`
	Verify      string `long:"verify-sha256" description:"verify the downloaded asset checksum against the one provided"`
	Version     bool   `short:"v" long:"version" description:"show version information"`
	Help        bool   `short:"h" long:"help" description:"show this help message"`
}
