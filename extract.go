package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

func NewExtractor(filename string, chooser Chooser) Extractor {
	gunzipper := func(r io.Reader) (io.Reader, error) {
		return gzip.NewReader(r)
	}
	b2unzipper := func(r io.Reader) (io.Reader, error) {
		return bzip2.NewReader(r), nil
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
	case strings.HasSuffix(filename, ".tar.bzip2"):
		return &TarExtractor{
			File:       chooser,
			Decompress: b2unzipper,
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
			Name:       filename,
			Decompress: gunzipper,
		}
	case strings.HasSuffix(filename, ".bzip2"):
		return &SingleFileExtractor{
			Name:       filename,
			Decompress: b2unzipper,
		}
	default:
		return &SingleFileExtractor{
			Name:       filename,
			Decompress: nounzipper,
		}
	}
}

type Extractor interface {
	Extract(data []byte) (*ExtractedFile, error)
}

type ExtractedFile struct {
	Name string
	Mode fs.FileMode
	Data []byte
}

type Chooser interface {
	Choose(name string, mode fs.FileMode) bool
}

type TarExtractor struct {
	File       Chooser
	Decompress func(r io.Reader) (io.Reader, error)
}

func (t *TarExtractor) Extract(data []byte) (*ExtractedFile, error) {
	r := bytes.NewReader(data)
	dr, err := t.Decompress(r)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(dr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar extract: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if t.File.Choose(hdr.Name, fs.FileMode(hdr.Mode)) {
				data, err := io.ReadAll(tr)
				return &ExtractedFile{
					Name: hdr.Name,
					Mode: fs.FileMode(hdr.Mode),
					Data: data,
				}, err
			}
		}
	}
	return nil, fmt.Errorf("target file not found in archive")
}

type ZipExtractor struct {
	File Chooser
}

func (z *ZipExtractor) Extract(data []byte) (*ExtractedFile, error) {
	r := bytes.NewReader(data)
	zr, err := zip.NewReader(r, int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, f := range zr.File {
		if z.File.Choose(f.Name, f.Mode()) {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("zip extract: %w", err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			return &ExtractedFile{
				Name: f.Name,
				Mode: f.Mode(),
				Data: data,
			}, err
		}
	}
	return nil, fmt.Errorf("target file not found in archive")
}

type SingleFileExtractor struct {
	Name       string
	Decompress func(r io.Reader) (io.Reader, error)
}

func (sf *SingleFileExtractor) Extract(data []byte) (*ExtractedFile, error) {
	r := bytes.NewReader(data)
	dr, err := sf.Decompress(r)
	if err != nil {
		return nil, err
	}

	decdata, err := io.ReadAll(dr)
	return &ExtractedFile{
		Name: sf.Name,
		Mode: 0666,
		Data: decdata,
	}, err
}

type BinaryChooser struct{}

func (b *BinaryChooser) Choose(name string, mode fs.FileMode) bool {
	return !mode.IsDir() && isExecAny(mode.Perm())
}

func isExecAny(mode os.FileMode) bool {
	return mode&0111 != 0
}

type FileChooser struct {
	File *regexp.Regexp
}

func (f *FileChooser) Choose(name string, mode fs.FileMode) bool {
	return f.File.MatchString(name)
}

func NewLicenseChooser() *FileChooser {
	return &FileChooser{
		File: regexp.MustCompile(`LICENSE`),
	}
}

func NewReadmeChooser() *FileChooser {
	return &FileChooser{
		File: regexp.MustCompile(`README|readme`),
	}
}
