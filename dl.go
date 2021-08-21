package main

import (
	"io"
	"net/http"

	pb "github.com/schollz/progressbar/v3"
)

func Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// TODO: rate-limiting authorization
	// if req.URL.Scheme == "https" && req.Host == "api.github.com" && os.Getenv("GITHUB_TOKEN") != "" {
	// 	req.Header.Set("Authorization:", fmt.Sprintf("token %s", os.Getenv("GITHUB_TOKEN")))
	// }
	return http.DefaultClient.Do(req)
}

// Download the file at 'url' and write the http response body to 'out'. The
// 'getbar' function allows the caller to construct a progress bar given the
// size of the file being downloaded, and the download will write to the
// returned progress bar.
func Download(url string, out io.Writer, getbar func(size int64) *pb.ProgressBar) error {
	resp, err := Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bar := getbar(resp.ContentLength)
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	return err
}
