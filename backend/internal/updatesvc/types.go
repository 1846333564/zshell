package updatesvc

import "net/http"

type Service struct {
	client *http.Client
}

const updateDownloadAttempts = 3

type ProgressReporter func(ProgressEvent)

type ProgressEvent struct {
	Stage          string `json:"stage"`
	Percent        int    `json:"percent"`
	Message        string `json:"message"`
	Detail         string `json:"detail,omitempty"`
	CurrentVersion string `json:"currentVersion,omitempty"`
	LatestVersion  string `json:"latestVersion,omitempty"`
	ReleaseName    string `json:"releaseName,omitempty"`
	ReleaseURL     string `json:"releaseUrl,omitempty"`
	AssetName      string `json:"assetName,omitempty"`
	AssetURL       string `json:"assetUrl,omitempty"`
	LoadedBytes    int64  `json:"loadedBytes,omitempty"`
	TotalBytes     int64  `json:"totalBytes,omitempty"`
	Attempt        int    `json:"attempt,omitempty"`
	MaxAttempts    int    `json:"maxAttempts,omitempty"`
}

type CheckResult struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Available      bool   `json:"available"`
	ReleaseName    string `json:"releaseName"`
	ReleaseURL     string `json:"releaseUrl"`
	Notes          string `json:"notes"`
	AssetName      string `json:"assetName"`
	AssetURL       string `json:"assetUrl"`
	PublishedAt    string `json:"publishedAt"`
	Verified       bool   `json:"verified"`
}

type ApplyResult struct {
	OK             bool   `json:"ok"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Message        string `json:"message"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
	Size               int64  `json:"size"`
}
