package main

import (
	"io"
	"net/http"

	pb "github.com/schollz/progressbar/v3"
)

func Download(url string, out io.Writer, getbar func(size int64) *pb.ProgressBar) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bar := getbar(resp.ContentLength)
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	return err
}
