package main

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
	"github.com/zyedidia/glob"
)

// An Extractor reads in some archive data and extracts a particular file from
// it. If there are multiple candidates it returns a list and an error
// explaining what happened.
type Extractor interface {
	Extract(data []byte, multiple bool) (ExtractedFile, []ExtractedFile, error)
}

// An ExtractedFile contains the data, name, and permissions of a file in the
// archive.
type ExtractedFile struct {
	Name        string // name to extract to
	ArchiveName string // name in archive
	mode        fs.FileMode
	Extract     func(to string) error
}

// Mode returns the filemode of the extracted file.
func (e ExtractedFile) Mode() fs.FileMode {
	return modeFrom(e.Name, e.mode)
}

func modeFrom(fname string, mode fs.FileMode) fs.FileMode {
	if isExec(fname, mode) {
		return mode | 0111
	}
	return mode
}

// String returns the archive name of this extracted file
func (e ExtractedFile) String() string {
	return e.ArchiveName
}

// A Chooser selects a file. It may list the file as a direct match (should be
// immediately extracted if found), or a possible match (only extract if it is
// the only match, or if the user manually requests it).
type Chooser interface {
	Choose(name string, mode fs.FileMode) (direct bool, possible bool)
}

// NewExtractor constructs an extractor for the given archive file using the
// given chooser. It will construct extractors for files ending in '.tar.gz',
// '.tar.bz2', '.tar', '.zip'. After these matches, if the file ends with
// '.gz', '.bz2' it will be decompressed and copied. Other files will simply
// be copied without any decompression or extraction.
func NewExtractor(filename string, tool string, chooser Chooser) Extractor {
	if tool == "" {
		tool = filename
	}

	gunzipper := func(r io.Reader) (io.Reader, error) {
		return gzip.NewReader(r)
	}
	b2unzipper := func(r io.Reader) (io.Reader, error) {
		return bzip2.NewReader(r), nil
	}
	xunzipper := func(r io.Reader) (io.Reader, error) {
		return xz.NewReader(bufio.NewReader(r))
	}
	nounzipper := func(r io.Reader) (io.Reader, error) {
		return r, nil
	}

	switch {
	case strings.HasSuffix(filename, ".tar.gz"), strings.HasSuffix(filename, ".tgz"):
		return &ArchiveExtractor{
			File:       chooser,
			Ar:         NewTarArchive,
			Decompress: gunzipper,
		}
	case strings.HasSuffix(filename, ".tar.bz2"):
		return &ArchiveExtractor{
			File:       chooser,
			Ar:         NewTarArchive,
			Decompress: b2unzipper,
		}
	case strings.HasSuffix(filename, ".tar.xz"):
		return &ArchiveExtractor{
			File:       chooser,
			Ar:         NewTarArchive,
			Decompress: xunzipper,
		}
	case strings.HasSuffix(filename, ".tar"):
		return &ArchiveExtractor{
			File:       chooser,
			Ar:         NewTarArchive,
			Decompress: nounzipper,
		}
	case strings.HasSuffix(filename, ".zip"):
		return &ArchiveExtractor{
			Ar:   NewZipArchive,
			File: chooser,
		}
	case strings.HasSuffix(filename, ".gz"):
		return &SingleFileExtractor{
			Rename:     tool,
			Name:       filename,
			Decompress: gunzipper,
		}
	case strings.HasSuffix(filename, ".bz2"):
		return &SingleFileExtractor{
			Rename:     tool,
			Name:       filename,
			Decompress: b2unzipper,
		}
	case strings.HasSuffix(filename, ".xz"):
		return &SingleFileExtractor{
			Rename:     tool,
			Name:       filename,
			Decompress: xunzipper,
		}
	default:
		return &SingleFileExtractor{
			Rename:     tool,
			Name:       filename,
			Decompress: nounzipper,
		}
	}
}

type ArchiveFn func(data []byte, decomp DecompFn) (Archive, error)
type DecompFn func(r io.Reader) (io.Reader, error)

type ArchiveExtractor struct {
	File       Chooser
	Ar         ArchiveFn
	Decompress DecompFn
}

func (a *ArchiveExtractor) Extract(data []byte, multiple bool) (ExtractedFile, []ExtractedFile, error) {
	var candidates []ExtractedFile

	ar, err := a.Ar(data, a.Decompress)
	if err != nil {
		return ExtractedFile{}, nil, err
	}
	for {
		f, err := ar.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ExtractedFile{}, nil, fmt.Errorf("extract: %w", err)
		}
		if f.Dir {
			continue
		}
		direct, possible := a.File.Choose(f.Name, f.Mode)
		if direct || possible {
			name := rename(f.Name, f.Name)

			data, err := ar.ReadAll()
			if err != nil {
				return ExtractedFile{}, nil, fmt.Errorf("extract: %w", err)
			}

			extract := func(to string) error {
				return writeFile(data, to, modeFrom(name, f.Mode))
			}

			ef := ExtractedFile{
				Name:        name,
				ArchiveName: f.Name,
				mode:        f.Mode,
				Extract:     extract,
			}
			if direct && !multiple {
				return ef, nil, err
			}
			if err == nil {
				candidates = append(candidates, ef)
			}
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil, nil
	} else if len(candidates) == 0 {
		return ExtractedFile{}, candidates, fmt.Errorf("target %v not found in archive", a.File)
	}
	return ExtractedFile{}, candidates, fmt.Errorf("%d candidates for target %v found", len(candidates), a.File)
}

// SingleFileExtractor extracts files called 'Name' after decompressing the
// file with 'Decompress'.
type SingleFileExtractor struct {
	Rename     string
	Name       string
	Decompress func(r io.Reader) (io.Reader, error)
}

func (sf *SingleFileExtractor) Extract(data []byte, multiple bool) (ExtractedFile, []ExtractedFile, error) {
	name := rename(sf.Name, sf.Rename)
	return ExtractedFile{
		Name:        name,
		ArchiveName: sf.Name,
		mode:        0666,
		Extract: func(to string) error {
			r := bytes.NewReader(data)
			dr, err := sf.Decompress(r)
			if err != nil {
				return err
			}

			decdata, err := io.ReadAll(dr)
			if err != nil {
				return err
			}
			return writeFile(decdata, to, modeFrom(name, 0666))
		},
	}, nil, nil
}

// attempt to rename 'file' to an appropriate executable name
func rename(file string, nameguess string) string {
	if isDefinitelyNotExec(file) {
		return file
	}

	var rename string
	if strings.HasSuffix(file, ".appimage") {
		// remove the .appimage extension
		rename = file[:len(file)-len(".appimage")]
	} else if strings.HasSuffix(file, ".exe") {
		// directly use xxx.exe
		rename = file
	} else {
		// otherwise use the rename guess
		rename = nameguess
	}
	return rename
}

// A BinaryChooser selects executable files. If the executable file has the
// name 'Tool' it is considered a direct match. If the file is only executable,
// it is a possible match.
type BinaryChooser struct {
	Tool string
}

func (b *BinaryChooser) Choose(name string, mode fs.FileMode) (bool, bool) {
	fmatch := filepath.Base(name) == b.Tool ||
		filepath.Base(name) == b.Tool+".exe" ||
		filepath.Base(name) == b.Tool+".appimage"

	possible := !mode.IsDir() && isExec(name, mode.Perm())
	return fmatch && possible, possible
}

func (b *BinaryChooser) String() string {
	return fmt.Sprintf("exe `%s`", b.Tool)
}

func isDefinitelyNotExec(file string) bool {
	// file is definitely not executable if it is .deb, .1, or .txt
	return strings.HasSuffix(file, ".deb") || strings.HasSuffix(file, ".1") ||
		strings.HasSuffix(file, ".txt")
}

func isExec(file string, mode os.FileMode) bool {
	if isDefinitelyNotExec(file) {
		return false
	}

	// file is executable if it is one of the following:
	// *.exe, *.appimage, no extension, executable file permissions
	return strings.HasSuffix(file, ".exe") ||
		strings.HasSuffix(file, ".appimage") ||
		!strings.Contains(file, ".") ||
		mode&0111 != 0
}

// LiteralFileChooser selects files with the name 'File'.
type LiteralFileChooser struct {
	File string
}

func (lf *LiteralFileChooser) Choose(name string, mode fs.FileMode) (bool, bool) {
	return false, filepath.Base(name) == filepath.Base(lf.File) && strings.HasSuffix(name, lf.File)
}

func (lf *LiteralFileChooser) String() string {
	return fmt.Sprintf("`%s`", lf.File)
}

type GlobChooser struct {
	expr string
	g    *glob.Glob
}

func NewGlobChooser(gl string) (*GlobChooser, error) {
	g, err := glob.Compile(gl)
	return &GlobChooser{
		g:    g,
		expr: gl,
	}, err
}

func (gc *GlobChooser) Choose(name string, mode fs.FileMode) (bool, bool) {
	return false, gc.g.MatchString(filepath.Base(name)) || gc.g.MatchString(name)
}

func (gc *GlobChooser) String() string {
	return fmt.Sprintf("`%s`", gc.expr)
}
