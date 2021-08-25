package main

import (
	"archive/tar"
	"archive/zip"
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
)

// An Extractor reads in some archive data and extracts a particular file from
// it. If there are multiple candidates it returns a list and an error
// explaining what happened.
type Extractor interface {
	Extract(data []byte) (ExtractedFile, []ExtractedFile, error)
}

// An ExtractedFile contains the data, name, and permissions of a file in the
// archive.
type ExtractedFile struct {
	Name        string // name to extract to
	ArchiveName string // name in archive
	mode        fs.FileMode
	Data        []byte
}

// Mode returns the filemode of the extracted file.
func (e ExtractedFile) Mode() fs.FileMode {
	if isExec(e.Name, e.mode) {
		return e.mode | 0111
	}
	return e.mode
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
	case strings.HasSuffix(filename, ".tar.gz"):
		return &TarExtractor{
			File:       chooser,
			Decompress: gunzipper,
		}
	case strings.HasSuffix(filename, ".tar.bz2"):
		return &TarExtractor{
			File:       chooser,
			Decompress: b2unzipper,
		}
	case strings.HasSuffix(filename, ".tar.xz"):
		return &TarExtractor{
			File:       chooser,
			Decompress: xunzipper,
		}
	case strings.HasSuffix(filename, ".tar"):
		return &TarExtractor{
			File:       chooser,
			Decompress: nounzipper,
		}
	case strings.HasSuffix(filename, ".zip"):
		return &ZipExtractor{
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

// TarExtractor extracts files matched by 'File' in a tar archive. It first
// decompresses the archive using the 'Decompress. function.
type TarExtractor struct {
	File       Chooser
	Decompress func(r io.Reader) (io.Reader, error)
}

func (t *TarExtractor) Extract(data []byte) (ExtractedFile, []ExtractedFile, error) {
	var candidates []ExtractedFile

	r := bytes.NewReader(data)
	dr, err := t.Decompress(r)
	if err != nil {
		return ExtractedFile{}, nil, err
	}
	tr := tar.NewReader(dr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ExtractedFile{}, nil, fmt.Errorf("tar extract: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			direct, possible := t.File.Choose(hdr.Name, fs.FileMode(hdr.Mode))
			if direct || possible {
				data, err := io.ReadAll(tr)
				f := ExtractedFile{
					Name:        rename(hdr.Name, hdr.Name),
					ArchiveName: hdr.Name,
					mode:        fs.FileMode(hdr.Mode),
					Data:        data,
				}
				if direct {
					return f, nil, err
				}
				if err == nil {
					candidates = append(candidates, f)
				}
			}
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil, nil
	} else if len(candidates) == 0 {
		return ExtractedFile{}, candidates, fmt.Errorf("target %v not found in archive", t.File)
	}
	return ExtractedFile{}, candidates, fmt.Errorf("%d candidates for target %v found", len(candidates), t.File)
}

// A ZipExtractor extracts files chosen by 'File' from a zip archive.
type ZipExtractor struct {
	File Chooser
}

func (z *ZipExtractor) Extract(data []byte) (ExtractedFile, []ExtractedFile, error) {
	var candidates []ExtractedFile

	r := bytes.NewReader(data)
	zr, err := zip.NewReader(r, int64(len(data)))
	if err != nil {
		return ExtractedFile{}, nil, fmt.Errorf("zip extract: %w", err)
	}

	for _, f := range zr.File {
		direct, possible := z.File.Choose(f.Name, f.Mode())
		if direct || possible {
			rc, err := f.Open()
			if err != nil {
				return ExtractedFile{}, nil, fmt.Errorf("zip extract: %w", err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			f := ExtractedFile{
				Name:        rename(f.Name, f.Name),
				ArchiveName: f.Name,
				mode:        f.Mode(),
				Data:        data,
			}
			if direct {
				return f, nil, err
			}
			if err == nil {
				candidates = append(candidates, f)
			}
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil, nil
	} else if len(candidates) == 0 {
		return ExtractedFile{}, candidates, fmt.Errorf("target %v not found in archive", z.File)
	}
	return ExtractedFile{}, candidates, fmt.Errorf("%d candidates for target %v found", len(candidates), z.File)
}

// SingleFileExtractor extracts files called 'Name' after decompressing the
// file with 'Decompress'.
type SingleFileExtractor struct {
	Rename     string
	Name       string
	Decompress func(r io.Reader) (io.Reader, error)
}

func (sf *SingleFileExtractor) Extract(data []byte) (ExtractedFile, []ExtractedFile, error) {
	r := bytes.NewReader(data)
	dr, err := sf.Decompress(r)
	if err != nil {
		return ExtractedFile{}, nil, err
	}

	decdata, err := io.ReadAll(dr)
	return ExtractedFile{
		Name:        rename(sf.Name, sf.Rename),
		ArchiveName: sf.Name,
		mode:        0666,
		Data:        decdata,
	}, nil, err
}

// attempt to rename 'file' to an appropriate executable name
func rename(file string, nameguess string) string {
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

func isExec(file string, mode os.FileMode) bool {
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
	return false, filepath.Base(name) == lf.File
}

func (lf *LiteralFileChooser) String() string {
	return fmt.Sprintf("`%s`", lf.File)
}
