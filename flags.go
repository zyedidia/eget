package main

var opts struct {
	Tag         string `short:"t" long:"tag" description:"tagged release to use instead of latest"`
	Output      string `long:"to" description:"extract to directory"`
	Yes         bool   `short:"y" description:"automatically approve all yes/no prompts"`
	System      string `short:"s" long:"system" description:"target system to download for"`
	ExtractFile string `short:"f" long:"file" description:"file name to extract"`
	Quiet       bool   `short:"q" long:"quiet" description:"only print essential output (prompts)"`
	Version     bool   `short:"v" long:"version" description:"show version information"`
	Help        bool   `short:"h" long:"help" description:"Show this help message"`
}
