package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// A Finder returns a list of URLs making up a project's assets.
type Finder interface {
	Find() ([]string, error)
}

// A GithubRelease matches the Assets portion of Github's release API json.
type GithubRelease struct {
	Assets []struct {
		DownloadURL string `json:"browser_download_url"`
	} `json:"assets"`

	Prerelease bool   `json:"prerelease"`
	Tag        string `json:"tag_name"`
}

type GithubError struct {
	Code   int
	Status string
	Body   []byte
	Url    string
}
type errResponse struct {
	Message string `json:"message"`
	Doc     string `json:"documentation_url"`
}

func (ge *GithubError) Error() string {
	var msg errResponse
	json.Unmarshal(ge.Body, &msg)

	if ge.Code == http.StatusForbidden {
		return fmt.Sprintf("%s: %s: %s", ge.Status, msg.Message, msg.Doc)
	}
	return fmt.Sprintf("%s (URL: %s)", ge.Status, ge.Url)
}

// A GithubAssetFinder finds assets for the given Repo at the given tag. Tags
// must be given as 'tag/<tag>'. Use 'latest' to get the latest release.
type GithubAssetFinder struct {
	Repo       string
	Tag        string
	Prerelease bool
}

func (f *GithubAssetFinder) Find() ([]string, error) {
	if f.Prerelease && f.Tag == "latest" {
		tag, err := f.getLatestTag()
		if err != nil {
			return nil, err
		}
		f.Tag = fmt.Sprintf("tags/%s", tag)
	}

	// query github's API for this repo/tag pair.
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/%s", f.Repo, f.Tag)
	resp, err := Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, &GithubError{
			Status: resp.Status,
			Code:   resp.StatusCode,
			Body:   body,
			Url:    url,
		}
	}

	// read and unmarshal the resulting json
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release GithubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		return nil, err
	}

	// accumulate all assets from the json into a slice
	assets := make([]string, 0, len(release.Assets))
	for _, a := range release.Assets {
		assets = append(assets, a.DownloadURL)
	}

	return assets, nil
}

// finds the latest pre-release and returns the tag
func (f *GithubAssetFinder) getLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", f.Repo)
	resp, err := Get(url)
	if err != nil {
		return "", fmt.Errorf("pre-release finder: %w", err)
	}

	var releases []GithubRelease

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("pre-release finder: %w", err)
	}
	err = json.Unmarshal(body, &releases)
	if err != nil {
		return "", fmt.Errorf("pre-release finder: %w", err)
	}

	if len(releases) <= 0 {
		return "", fmt.Errorf("no releases found")
	}

	return releases[0].Tag, nil
}

// A DirectAssetFinder returns the embedded URL directly as the only asset.
type DirectAssetFinder struct {
	URL string
}

func (f *DirectAssetFinder) Find() ([]string, error) {
	return []string{f.URL}, nil
}
