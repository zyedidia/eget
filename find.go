package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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

	Prerelease bool      `json:"prerelease"`
	Tag        string    `json:"tag_name"`
	CreatedAt  time.Time `json:"created_at"`
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

const (
	Github = "github"
	Gitea  = "gitea"
)

func (ge *GithubError) Error() string {
	var msg errResponse
	json.Unmarshal(ge.Body, &msg)

	if ge.Code == http.StatusForbidden {
		return fmt.Sprintf("%s: %s: %s", ge.Status, msg.Message, msg.Doc)
	}
	return fmt.Sprintf("%s (URL: %s)", ge.Status, ge.Url)
}

// A AssetFinder finds assets for the given Repo at the given tag. Tags
// must be given as 'tag/<tag>'. Use 'latest' to get the latest release.
type AssetFinder struct {
	Repo       string
	Tag        string
	Prerelease bool
	HostType   string
	Host       string
	MinTime    time.Time // release must be after MinTime to be found
}

var ErrNoUpgrade = errors.New("requested release is not more recent than current version")

func (f *AssetFinder) Find() ([]string, error) {
	if f.Prerelease && f.Tag == "latest" {
		tag, err := f.getLatestTag()
		if err != nil {
			return nil, err
		}
		f.Tag = fmt.Sprintf("tags/%s", tag)
	}

	// query github's API for this repo/tag pair.
	url := fmt.Sprintf("%s/%s", f.getReleasesUrl(), f.Tag)
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
		if strings.HasPrefix(f.Tag, "tags/") && resp.StatusCode == http.StatusNotFound {
			return f.FindMatch()
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

	if release.CreatedAt.Before(f.MinTime) {
		return nil, ErrNoUpgrade
	}

	// accumulate all assets from the json into a slice
	assets := make([]string, 0, len(release.Assets))
	for _, a := range release.Assets {
		assets = append(assets, a.DownloadURL)
	}

	return assets, nil
}

func (f *AssetFinder) FindMatch() ([]string, error) {
	tag := f.Tag[len("tags/"):]
	url := f.getReleasesUrl()

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s?page=%d", url, page)
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

		var releases []GithubRelease
		err = json.Unmarshal(body, &releases)
		if err != nil {
			return nil, err
		}

		for _, r := range releases {
			if !f.Prerelease && r.Prerelease {
				continue
			}
			if strings.Contains(r.Tag, tag) && !r.CreatedAt.Before(f.MinTime) {
				// we have a winner
				assets := make([]string, 0, len(r.Assets))
				for _, a := range r.Assets {
					assets = append(assets, a.DownloadURL)
				}
				return assets, nil
			}
		}

		if len(releases) < 30 {
			break
		}
	}

	return nil, fmt.Errorf("no matching tag for '%s'", tag)
}

func (f *AssetFinder) getReleasesUrl() string {
	switch f.HostType {
	case Github:
		return fmt.Sprintf("https://api.%s/repos/%s/releases", f.Host, f.Repo)
	case Gitea:
		return fmt.Sprintf("https://%s/api/v1/repos/%s/releases", f.Host, f.Repo)
	default:
		return ""
	}
}

// finds the latest pre-release and returns the tag
func (f *AssetFinder) getLatestTag() (string, error) {
	url := f.getReleasesUrl()
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

type GithubSourceFinder struct {
	Host string
	Tool string
	Repo string
	Tag  string
}

func (f *GithubSourceFinder) Find() ([]string, error) {
	return []string{fmt.Sprintf("https://%s/%s/tarball/%s/%s.tar.gz", f.Host, f.Repo, f.Tag, f.Tool)}, nil
}
