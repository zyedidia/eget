package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

type FullReader func() ([]byte, error)

type File struct {
	Name    string
	Mode    fs.FileMode
	ReadAll FullReader
	Dir     bool
}

type Archive interface {
	Next() (File, error)
}

type TarArchive struct {
	r *tar.Reader
}

func NewTarArchive(data []byte, decompress DecompFn) (Archive, error) {
	r := bytes.NewReader(data)
	dr, err := decompress(r)
	if err != nil {
		return nil, err
	}
	return &TarArchive{
		r: tar.NewReader(dr),
	}, nil
}

func (t *TarArchive) Next() (File, error) {
	for {
		hdr, err := t.r.Next()
		if err != nil {
			return File{}, err
		}
		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeDir {
			return File{
				Name: hdr.Name,
				Mode: fs.FileMode(hdr.Mode),
				ReadAll: func() ([]byte, error) {
					return io.ReadAll(t.r)
				},
				Dir: hdr.Typeflag == tar.TypeDir,
			}, err
		}
	}
}

type ZipArchive struct {
	r   *zip.Reader
	idx int
}

// decompressor does nothing for a zip archive because it already has built-in
// compression.
func NewZipArchive(data []byte, d DecompFn) (Archive, error) {
	r := bytes.NewReader(data)
	zr, err := zip.NewReader(r, int64(len(data)))
	return &ZipArchive{
		r:   zr,
		idx: 0,
	}, err
}

func (z *ZipArchive) Next() (File, error) {
	if z.idx < 0 || z.idx >= len(z.r.File) {
		return File{}, io.EOF
	}

	f := z.r.File[z.idx]

	return File{
		Name: f.Name,
		Mode: f.Mode(),
		ReadAll: func() ([]byte, error) {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("zip extract: %w", err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			return data, err
		},
		Dir: strings.HasSuffix(f.Name, "/"),
	}, nil
}
