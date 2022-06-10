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

type FileType byte

const (
	TypeNormal FileType = iota
	TypeDir
	TypeLink
	TypeSymlink
	TypeOther
)

func tarft(typ byte) FileType {
	switch typ {
	case tar.TypeReg:
		return TypeNormal
	case tar.TypeDir:
		return TypeDir
	case tar.TypeLink:
		return TypeLink
	case tar.TypeSymlink:
		return TypeSymlink
	}
	return TypeOther
}

type File struct {
	Name     string
	LinkName string
	Mode     fs.FileMode
	Type     FileType
}

func (f File) Dir() bool {
	return f.Type == TypeDir
}

type Archive interface {
	Next() (File, error)
	ReadAll() ([]byte, error)
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
		ft := tarft(hdr.Typeflag)
		if ft != TypeOther {
			return File{
				Name:     hdr.Name,
				LinkName: hdr.Linkname,
				Mode:     fs.FileMode(hdr.Mode),
				Type:     ft,
			}, err
		}
	}
}

func (t *TarArchive) ReadAll() ([]byte, error) {
	return io.ReadAll(t.r)
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
		idx: -1,
	}, err
}

func (z *ZipArchive) Next() (File, error) {
	z.idx++

	if z.idx < 0 || z.idx >= len(z.r.File) {
		return File{}, io.EOF
	}

	f := z.r.File[z.idx]

	typ := TypeNormal
	if strings.HasSuffix(f.Name, "/") {
		typ = TypeDir
	}

	return File{
		Name: f.Name,
		Mode: f.Mode(),
		Type: typ,
	}, nil
}

func (z *ZipArchive) ReadAll() ([]byte, error) {
	if z.idx < 0 || z.idx >= len(z.r.File) {
		return nil, io.EOF
	}
	f := z.r.File[z.idx]
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("zip extract: %w", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	return data, err
}
