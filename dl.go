package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	pb "github.com/schollz/progressbar/v3"
	"github.com/zyedidia/eget/home"
)

func tokenFrom(s string) (string, error) {
	if strings.HasPrefix(s, "@") {
		f, err := home.Expand(s[1:])
		if err != nil {
			return "", err
		}
		b, err := os.ReadFile(f)
		return strings.TrimRight(string(b), "\r\n"), nil
	}
	return s, nil
}

var ErrNoToken = errors.New("no github token")

func getGithubToken() (string, error) {
	if os.Getenv("EGET_GITHUB_TOKEN") != "" {
		return tokenFrom(os.Getenv("EGET_GITHUB_TOKEN"))
	}
	if os.Getenv("GITHUB_TOKEN") != "" {
		return tokenFrom(os.Getenv("GITHUB_TOKEN"))
	}
	return "", ErrNoToken
}

func SetAuthHeader(req *http.Request) *http.Request {
	token, err := getGithubToken()
	if err != nil && !errors.Is(err, ErrNoToken) {
		fmt.Fprintln(os.Stderr, "warning: not using github token:", err)
	}

	if req.URL.Scheme == "https" && req.Host == "api.github.com" && err == nil {
		if opts.DisableSSL {
			fmt.Fprintln(os.Stderr, "error: cannot use GitHub token if SSL verification is disabled")
			os.Exit(1)
		}
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	return req
}

func Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req = SetAuthHeader(req)

	proxyClient := &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.DisableSSL},
	}}

	return proxyClient.Do(req)
}

type RateLimitJson struct {
	Resources map[string]RateLimit
}

type RateLimit struct {
	Limit     int
	Remaining int
	Reset     int64
}

func (r RateLimit) ResetTime() time.Time {
	return time.Unix(r.Reset, 0)
}

func (r RateLimit) String() string {
	now := time.Now()
	rtime := r.ResetTime()
	if rtime.Before(now) {
		return fmt.Sprintf("Limit: %d, Remaining: %d, Reset: %v", r.Limit, r.Remaining, rtime)
	} else {
		return fmt.Sprintf(
			"Limit: %d, Remaining: %d, Reset: %v (%v)",
			r.Limit, r.Remaining, rtime, rtime.Sub(now).Round(time.Second),
		)
	}
}

func GetRateLimit() (RateLimit, error) {
	url := "https://api.github.com/rate_limit"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return RateLimit{}, err
	}

	req = SetAuthHeader(req)

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return RateLimit{}, err
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return RateLimit{}, err
	}

	var parsed RateLimitJson
	err = json.Unmarshal(b, &parsed)

	return parsed.Resources["core"], err
}

// Download the file at 'url' and write the http response body to 'out'. The
// 'getbar' function allows the caller to construct a progress bar given the
// size of the file being downloaded, and the download will write to the
// returned progress bar.
func Download(url string, out io.Writer, getbar func(size int64) *pb.ProgressBar) error {
	if IsLocalFile(url) {
		f, err := os.Open(url)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(out, f)
		return err
	}

	resp, err := Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bar := getbar(resp.ContentLength)
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	return err
}
