package updatesvc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"zshell/backend/internal/appinfo"
)

func (s *Service) latestRelease(ctx context.Context, report ProgressReporter) (githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", appinfo.GitHubOwner, appinfo.GitHubRepo)
	reportProgress(report, ProgressEvent{
		Stage:          "checking",
		Percent:        8,
		Message:        "正在请求 GitHub Release API",
		Detail:         url,
		CurrentVersion: appinfo.Version,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", appinfo.ProductName+"/"+appinfo.Version)

	resp, err := s.client.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("check github release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		reportProgress(report, ProgressEvent{
			Stage:   "checking",
			Percent: 12,
			Message: "API 未找到 latest，改用 Release 页面解析",
			Detail:  resp.Status,
		})
		return s.latestReleaseByRedirect(ctx, report)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			reportProgress(report, ProgressEvent{
				Stage:   "checking",
				Percent: 12,
				Message: "GitHub API 受限，改用 Release 页面解析",
				Detail:  resp.Status,
			})
			release, fallbackErr := s.latestReleaseByRedirect(ctx, report)
			if fallbackErr == nil {
				return release, nil
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			return githubRelease{}, fmt.Errorf("github api limited: %s %s; release page fallback failed: %w", resp.Status, strings.TrimSpace(string(body)), fallbackErr)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return githubRelease{}, fmt.Errorf("github release request failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("decode github release: %w", err)
	}
	reportProgress(report, ProgressEvent{
		Stage:         "checking",
		Percent:       16,
		Message:       "GitHub Release 信息读取完成",
		LatestVersion: normalizeVersion(release.TagName),
		ReleaseName:   release.Name,
		ReleaseURL:    release.HTMLURL,
	})
	return release, nil
}

func (s *Service) latestReleaseByRedirect(ctx context.Context, report ProgressReporter) (githubRelease, error) {
	url := fmt.Sprintf("https://github.com/%s/%s/releases/latest", appinfo.GitHubOwner, appinfo.GitHubRepo)
	reportProgress(report, ProgressEvent{
		Stage:   "checking",
		Percent: 14,
		Message: "正在打开 GitHub Release latest 页面",
		Detail:  url,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("User-Agent", appinfo.ProductName+"/"+appinfo.Version)

	resp, err := s.client.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("check github release page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubRelease{}, fmt.Errorf("github release page request failed: %s", resp.Status)
	}

	parts := strings.Split(strings.Trim(resp.Request.URL.Path, "/"), "/")
	if len(parts) < 5 {
		return githubRelease{}, fmt.Errorf("cannot resolve latest release tag")
	}
	tag := parts[len(parts)-1]
	version := normalizeVersion(tag)
	if version == "" {
		return githubRelease{}, fmt.Errorf("latest release tag is empty")
	}

	assetName := appinfo.ReleaseAssetName(version)
	downloadBase := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", appinfo.GitHubOwner, appinfo.GitHubRepo, tag)
	release := githubRelease{
		TagName: tag,
		Name:    appinfo.ProductName + " " + version,
		HTMLURL: resp.Request.URL.String(),
		Assets: []githubAsset{
			{
				Name:               assetName,
				BrowserDownloadURL: downloadBase + "/" + assetName,
			},
			{
				Name:               assetName + ".sha256",
				BrowserDownloadURL: downloadBase + "/" + assetName + ".sha256",
			},
		},
	}
	reportProgress(report, ProgressEvent{
		Stage:         "checking",
		Percent:       16,
		Message:       "Release 页面解析完成",
		LatestVersion: version,
		ReleaseName:   release.Name,
		ReleaseURL:    release.HTMLURL,
	})
	return release, nil
}
