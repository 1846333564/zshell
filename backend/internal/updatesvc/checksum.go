package updatesvc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"wiShell/backend/internal/appinfo"
)

func assetDigest(asset githubAsset) string {
	digest := strings.TrimSpace(asset.Digest)
	if strings.HasPrefix(strings.ToLower(digest), "sha256:") {
		return strings.TrimSpace(digest[len("sha256:"):])
	}
	return ""
}

func hasSHA256Asset(assets []githubAsset, exeName string) bool {
	candidates := []string{exeName + ".sha256", "sha256.txt"}
	for _, candidate := range candidates {
		for _, asset := range assets {
			if strings.EqualFold(asset.Name, candidate) && asset.BrowserDownloadURL != "" {
				return true
			}
		}
	}
	return false
}

func (s *Service) findSHA256Digest(ctx context.Context, assets []githubAsset, exeName string, report ProgressReporter) (string, error) {
	candidates := []string{exeName + ".sha256", "sha256.txt"}
	for _, candidate := range candidates {
		for _, asset := range assets {
			if strings.EqualFold(asset.Name, candidate) && asset.BrowserDownloadURL != "" {
				digest, err := s.downloadSHA256(ctx, asset.BrowserDownloadURL, exeName, report)
				if err != nil {
					return "", err
				}
				return digest, nil
			}
		}
	}
	return "", nil
}

func (s *Service) downloadSHA256(ctx context.Context, url string, exeName string, report ProgressReporter) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= updateDownloadAttempts; attempt++ {
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			return "", stopErr
		}
		reportProgress(report, ProgressEvent{
			Stage:       "checksum",
			Percent:     85,
			Message:     "正在下载校验文件",
			Detail:      url,
			AssetName:   exeName,
			AssetURL:    url,
			Attempt:     attempt,
			MaxAttempts: updateDownloadAttempts,
		})
		digest, err := s.downloadSHA256Once(ctx, url, exeName)
		if err == nil {
			reportProgress(report, ProgressEvent{
				Stage:       "checksum",
				Percent:     88,
				Message:     "校验文件读取完成",
				Detail:      digest,
				AssetName:   exeName,
				AssetURL:    url,
				Attempt:     attempt,
				MaxAttempts: updateDownloadAttempts,
			})
			return digest, nil
		}
		lastErr = err
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			return "", stopErr
		}
		reportProgress(report, ProgressEvent{
			Stage:       "retrying",
			Percent:     85,
			Message:     "校验文件下载失败，准备重试",
			Detail:      err.Error(),
			AssetName:   exeName,
			AssetURL:    url,
			Attempt:     attempt,
			MaxAttempts: updateDownloadAttempts,
		})
		waitBeforeRetry(ctx, attempt)
	}
	return "", explainDownloadError("下载校验文件失败", url, lastErr)
}

func (s *Service) downloadSHA256Once(ctx context.Context, url string, exeName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", appinfo.ProductName+"/"+appinfo.Version)

	resp, err := s.client.Do(req)
	if err != nil {
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			return "", stopErr
		}
		return "", fmt.Errorf("download checksum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download checksum failed: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			return "", stopErr
		}
		return "", fmt.Errorf("read checksum: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		if len(fields) > 1 && !strings.Contains(strings.Join(fields[1:], " "), exeName) {
			continue
		}
		digest := strings.TrimPrefix(strings.ToLower(fields[0]), "sha256:")
		if len(digest) == 64 {
			return digest, nil
		}
	}
	return "", fmt.Errorf("checksum asset does not contain a SHA256 digest for %s", exeName)
}
