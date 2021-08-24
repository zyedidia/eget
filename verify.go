package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

type Verifier interface {
	Verify(b []byte) error
}

type NoVerifier struct{}

func (n *NoVerifier) Verify(b []byte) error {
	return nil
}

type Sha256Error struct {
	Expected []byte
	Got      []byte
}

func (e *Sha256Error) Error() string {
	return fmt.Sprintf("sha256 checksum mismatch:\nexpected: %x\ngot:      %x", e.Expected, e.Got)
}

type Sha256Verifier struct {
	Expected []byte
}

func NewSha256Verifier(expectedHex string) (*Sha256Verifier, error) {
	expected, _ := hex.DecodeString(expectedHex)
	if len(expected) != sha256.Size {
		return nil, fmt.Errorf("sha256sum (%s) too small: %d bytes decoded", expectedHex, len(expectedHex))
	}
	return &Sha256Verifier{
		Expected: expected,
	}, nil
}

func (s256 *Sha256Verifier) Verify(b []byte) error {
	sum := sha256.Sum256(b)
	if bytes.Equal(sum[:], s256.Expected) {
		return nil
	}
	return &Sha256Error{
		Expected: s256.Expected,
		Got:      sum[:],
	}
}

type Sha256Printer struct{}

func (s256 *Sha256Printer) Verify(b []byte) error {
	sum := sha256.Sum256(b)
	fmt.Printf("%x\n", sum)
	return nil
}

type Sha256AssetVerifier struct {
	AssetURL string
}

func (s256 *Sha256AssetVerifier) Verify(b []byte) error {
	resp, err := Get(s256.AssetURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	expected := make([]byte, sha256.Size)
	n, err := hex.Decode(expected, data)
	if n < sha256.Size {
		return fmt.Errorf("sha256sum (%s) too small: %d bytes decoded", string(data), n)
	}
	sum := sha256.Sum256(b)
	if bytes.Equal(sum[:], expected[:n]) {
		return nil
	}
	return &Sha256Error{
		Expected: expected[:n],
		Got:      sum[:],
	}
}
