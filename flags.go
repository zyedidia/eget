package main

var opts struct {
	Tag         string `short:"t" long:"tag" description:"tagged release to use instead of latest"`
	Output      string `long:"to" description:"extract to directory"`
	System      string `short:"s" long:"system" description:"target system to download for"`
	ExtractFile string `short:"f" long:"file" description:"file name to extract"`
	Quiet       bool   `short:"q" long:"quiet" description:"only print essential output"`
	DLOnly      bool   `long:"download-only" description:"stop after downloading the asset (no extraction)"`
	URL         bool   `long:"url" description:"download from the given URL directly"`
	Asset       string `long:"asset" description:"download a specific asset"`
	Exec        bool   `short:"x" description:"force the extracted file to be executable"`
	Hash        bool   `long:"sha256" description:"show the SHA-256 hash of the downloaded asset"`
	Version     bool   `short:"v" long:"version" description:"show version information"`
	Help        bool   `short:"h" long:"help" description:"show this help message"`
}
