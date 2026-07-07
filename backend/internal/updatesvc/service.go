package updatesvc

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"wiShell/backend/internal/appinfo"
)

func NewService() *Service {
	return &Service{
		client: &http.Client{Timeout: 75 * time.Second},
	}
}

func (s *Service) Check(ctx context.Context) (CheckResult, error) {
	release, err := s.latestRelease(ctx, nil)
	if err != nil {
		return CheckResult{}, explainCheckError(err)
	}

	latestVersion := normalizeVersion(release.TagName)
	if latestVersion == "" {
		return CheckResult{}, fmt.Errorf("latest release tag is empty")
	}

	result := CheckResult{
		CurrentVersion: appinfo.Version,
		LatestVersion:  latestVersion,
		Available:      compareVersions(latestVersion, appinfo.Version) > 0,
		ReleaseName:    release.Name,
		ReleaseURL:     release.HTMLURL,
		Notes:          release.Body,
		PublishedAt:    release.PublishedAt,
	}

	asset := findExecutableAsset(release.Assets, latestVersion)
	if asset.Name != "" {
		result.AssetName = asset.Name
		result.AssetURL = asset.BrowserDownloadURL
		result.Verified = assetDigest(asset) != "" || hasSHA256Asset(release.Assets, asset.Name)
	}
	if result.Available && result.AssetURL == "" {
		return CheckResult{}, fmt.Errorf("latest release does not contain %s", appinfo.ReleaseAssetName(latestVersion))
	}
	return result, nil
}

func (s *Service) Apply(ctx context.Context) (ApplyResult, error) {
	return s.ApplyWithProgress(ctx, nil)
}

func (s *Service) ApplyWithProgress(ctx context.Context, report ProgressReporter) (ApplyResult, error) {
	reportProgress(report, ProgressEvent{
		Stage:          "checking",
		Percent:        4,
		Message:        "正在连接 GitHub Release",
		CurrentVersion: appinfo.Version,
	})

	release, err := s.latestRelease(ctx, report)
	if err != nil {
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			reportProgress(report, ProgressEvent{
				Stage:   "stopped",
				Percent: 0,
				Message: ErrStopped.Error(),
			})
			return ApplyResult{}, stopErr
		}
		return ApplyResult{}, explainCheckError(err)
	}

	latestVersion := normalizeVersion(release.TagName)
	reportProgress(report, ProgressEvent{
		Stage:          "version",
		Percent:        18,
		Message:        "已获取最新版本信息",
		CurrentVersion: appinfo.Version,
		LatestVersion:  latestVersion,
		ReleaseName:    release.Name,
		ReleaseURL:     release.HTMLURL,
	})
	if compareVersions(latestVersion, appinfo.Version) <= 0 {
		reportProgress(report, ProgressEvent{
			Stage:          "done",
			Percent:        100,
			Message:        "当前已经是最新版本",
			CurrentVersion: appinfo.Version,
			LatestVersion:  latestVersion,
			ReleaseURL:     release.HTMLURL,
		})
		return ApplyResult{
			OK:             true,
			CurrentVersion: appinfo.Version,
			LatestVersion:  latestVersion,
			Message:        "当前已经是最新版本",
		}, nil
	}

	asset := findExecutableAsset(release.Assets, latestVersion)
	if asset.Name == "" || asset.BrowserDownloadURL == "" {
		return ApplyResult{}, fmt.Errorf("latest release does not contain %s", appinfo.ReleaseAssetName(latestVersion))
	}
	reportProgress(report, ProgressEvent{
		Stage:          "asset",
		Percent:        24,
		Message:        "已定位更新安装包",
		CurrentVersion: appinfo.Version,
		LatestVersion:  latestVersion,
		ReleaseName:    release.Name,
		ReleaseURL:     release.HTMLURL,
		AssetName:      asset.Name,
		AssetURL:       asset.BrowserDownloadURL,
		TotalBytes:     asset.Size,
	})

	downloadPath, digest, err := s.downloadExecutable(ctx, asset, report)
	if err != nil {
		if IsStopped(err) {
			reportProgress(report, ProgressEvent{
				Stage:   "stopped",
				Percent: 0,
				Message: ErrStopped.Error(),
			})
		}
		return ApplyResult{}, err
	}
	if stopErr := stopIfCanceled(ctx); stopErr != nil {
		_ = os.Remove(downloadPath)
		reportProgress(report, ProgressEvent{
			Stage:   "stopped",
			Percent: 82,
			Message: ErrStopped.Error(),
		})
		return ApplyResult{}, stopErr
	}

	expectedDigest := assetDigest(asset)
	if expectedDigest == "" {
		reportProgress(report, ProgressEvent{
			Stage:         "checksum",
			Percent:       84,
			Message:       "正在获取 SHA256 校验文件",
			AssetName:     asset.Name,
			AssetURL:      asset.BrowserDownloadURL,
			LatestVersion: latestVersion,
		})
		expectedDigest, err = s.findSHA256Digest(ctx, release.Assets, asset.Name, report)
		if err != nil {
			_ = os.Remove(downloadPath)
			if IsStopped(err) {
				reportProgress(report, ProgressEvent{
					Stage:   "stopped",
					Percent: 84,
					Message: ErrStopped.Error(),
				})
			}
			return ApplyResult{}, err
		}
	}
	if stopErr := stopIfCanceled(ctx); stopErr != nil {
		_ = os.Remove(downloadPath)
		reportProgress(report, ProgressEvent{
			Stage:   "stopped",
			Percent: 88,
			Message: ErrStopped.Error(),
		})
		return ApplyResult{}, stopErr
	}
	if expectedDigest != "" && !strings.EqualFold(digest, expectedDigest) {
		_ = os.Remove(downloadPath)
		return ApplyResult{}, fmt.Errorf("downloaded update checksum mismatch")
	}
	reportProgress(report, ProgressEvent{
		Stage:         "checksum",
		Percent:       90,
		Message:       "安装包校验完成",
		Detail:        digest,
		AssetName:     asset.Name,
		LatestVersion: latestVersion,
	})
	if stopErr := stopIfCanceled(ctx); stopErr != nil {
		_ = os.Remove(downloadPath)
		reportProgress(report, ProgressEvent{
			Stage:   "stopped",
			Percent: 90,
			Message: ErrStopped.Error(),
		})
		return ApplyResult{}, stopErr
	}

	reportProgress(report, ProgressEvent{
		Stage:         "scheduling",
		Percent:       94,
		Message:       "正在写入替换脚本",
		LatestVersion: latestVersion,
	})
	if err := scheduleReplacement(downloadPath); err != nil {
		_ = os.Remove(downloadPath)
		return ApplyResult{}, err
	}

	reportProgress(report, ProgressEvent{
		Stage:          "restarting",
		Percent:        100,
		Message:        "更新已下载，应用即将重启",
		CurrentVersion: appinfo.Version,
		LatestVersion:  latestVersion,
		AssetName:      asset.Name,
	})
	go func() {
		time.Sleep(1800 * time.Millisecond)
		os.Exit(0)
	}()

	return ApplyResult{
		OK:             true,
		CurrentVersion: appinfo.Version,
		LatestVersion:  latestVersion,
		Message:        "更新已下载，应用即将重启",
	}, nil
}
