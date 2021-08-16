package main

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
)

type ExtractedFile struct {
	Name string
	Mode fs.FileMode
	Data []byte
}

type Chooser interface {
	Choose(name string, mode fs.FileMode) bool
}

type TarExtractor struct {
	File Chooser
}

func (t *TarExtractor) Extract(r io.Reader) (*ExtractedFile, error) {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar extract: %w", err)
		}
		if t.File.Choose(hdr.Name, fs.FileMode(hdr.Mode)) {
			data, err := io.ReadAll(tr)
			return &ExtractedFile{
				Name: hdr.Name,
				Mode: fs.FileMode(hdr.Mode),
				Data: data,
			}, err
		}
	}
	return nil, fmt.Errorf("target file not found in archive")
}

type ZipExtractor struct {
	File Chooser
}

func (z *ZipExtractor) Extract(r io.ReaderAt, size int64) (*ExtractedFile, error) {
	zr, err := zip.NewReader(r, size)
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

type BinaryChooser struct{}

func (b *BinaryChooser) Choose(name string, mode fs.FileMode) bool {
	return isExecAny(mode)
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
