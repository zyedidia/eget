package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// A Finder returns a list of URLs making up a project's assets.
type Finder interface {
	Find() ([]string, error)
}

type GithubRelease struct {
	Assets []struct {
		DownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type GithubAssetFinder struct {
	Repo string
	Tag  string
}

func (f *GithubAssetFinder) Find() ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/%s", f.Repo, f.Tag)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned bad status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release GithubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		return nil, err
	}

	assets := make([]string, 0, len(release.Assets))
	for _, a := range release.Assets {
		assets = append(assets, a.DownloadURL)
	}

	return assets, nil
}
