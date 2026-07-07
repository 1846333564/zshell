package updatesvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"wiShell/backend/internal/appinfo"
)

func (s *Service) downloadExecutable(ctx context.Context, asset githubAsset, report ProgressReporter) (string, string, error) {
	var lastErr error
	for attempt := 1; attempt <= updateDownloadAttempts; attempt++ {
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			return "", "", stopErr
		}
		reportProgress(report, ProgressEvent{
			Stage:       "downloading",
			Percent:     30,
			Message:     "开始下载更新安装包",
			Detail:      asset.BrowserDownloadURL,
			AssetName:   asset.Name,
			AssetURL:    asset.BrowserDownloadURL,
			TotalBytes:  asset.Size,
			Attempt:     attempt,
			MaxAttempts: updateDownloadAttempts,
		})
		target, digest, err := s.downloadExecutableOnce(ctx, asset, attempt, report)
		if err == nil {
			return target, digest, nil
		}
		lastErr = err
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			return "", "", stopErr
		}
		reportProgress(report, ProgressEvent{
			Stage:       "retrying",
			Percent:     30,
			Message:     "下载失败，准备重试",
			Detail:      err.Error(),
			AssetName:   asset.Name,
			AssetURL:    asset.BrowserDownloadURL,
			Attempt:     attempt,
			MaxAttempts: updateDownloadAttempts,
		})
		waitBeforeRetry(ctx, attempt)
	}
	return "", "", explainDownloadError("下载更新失败", asset.BrowserDownloadURL, lastErr)
}

func (s *Service) downloadExecutableOnce(ctx context.Context, asset githubAsset, attempt int, report ProgressReporter) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", appinfo.ProductName+"/"+appinfo.Version)

	resp, err := s.client.Do(req)
	if err != nil {
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			return "", "", stopErr
		}
		return "", "", fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("download update failed: %s", resp.Status)
	}

	totalBytes := resp.ContentLength
	if totalBytes <= 0 {
		totalBytes = asset.Size
	}
	reportProgress(report, ProgressEvent{
		Stage:       "downloading",
		Percent:     32,
		Message:     "下载连接已建立",
		Detail:      resp.Status,
		AssetName:   asset.Name,
		AssetURL:    asset.BrowserDownloadURL,
		TotalBytes:  totalBytes,
		Attempt:     attempt,
		MaxAttempts: updateDownloadAttempts,
	})

	dir := filepath.Join(os.TempDir(), "wiShell-update")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	target := filepath.Join(dir, asset.Name)
	file, err := os.Create(target)
	if err != nil {
		return "", "", fmt.Errorf("create update temp file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	progress := newDownloadProgressReader(resp.Body, totalBytes, func(loaded int64, total int64) {
		reportProgress(report, ProgressEvent{
			Stage:       "downloading",
			Percent:     downloadPercent(loaded, total),
			Message:     "正在下载更新安装包",
			AssetName:   asset.Name,
			AssetURL:    asset.BrowserDownloadURL,
			LoadedBytes: loaded,
			TotalBytes:  total,
			Attempt:     attempt,
			MaxAttempts: updateDownloadAttempts,
		})
	})
	if _, err := io.CopyBuffer(io.MultiWriter(file, hash), progress, make([]byte, 256*1024)); err != nil {
		_ = os.Remove(target)
		if stopErr := stopIfCanceled(ctx); stopErr != nil {
			return "", "", stopErr
		}
		return "", "", fmt.Errorf("write update temp file: %w", err)
	}
	reportProgress(report, ProgressEvent{
		Stage:       "downloading",
		Percent:     82,
		Message:     "更新安装包下载完成",
		AssetName:   asset.Name,
		AssetURL:    asset.BrowserDownloadURL,
		LoadedBytes: progress.loaded,
		TotalBytes:  totalBytes,
		Attempt:     attempt,
		MaxAttempts: updateDownloadAttempts,
	})

	return target, hex.EncodeToString(hash.Sum(nil)), nil
}

type downloadProgressReader struct {
	source     io.Reader
	total      int64
	loaded     int64
	lastReport time.Time
	report     func(loaded int64, total int64)
}

func newDownloadProgressReader(source io.Reader, total int64, report func(loaded int64, total int64)) *downloadProgressReader {
	return &downloadProgressReader{
		source: source,
		total:  total,
		report: report,
	}
}

func (r *downloadProgressReader) Read(p []byte) (int, error) {
	n, err := r.source.Read(p)
	if n > 0 {
		r.loaded += int64(n)
		r.maybeReport(false)
	}
	if err == io.EOF {
		r.maybeReport(true)
	}
	return n, err
}

func (r *downloadProgressReader) maybeReport(force bool) {
	if r.report == nil {
		return
	}
	now := time.Now()
	if !force && !r.lastReport.IsZero() && now.Sub(r.lastReport) < 180*time.Millisecond {
		return
	}
	r.lastReport = now
	r.report(r.loaded, r.total)
}

func downloadPercent(loaded int64, total int64) int {
	if total <= 0 {
		if loaded > 0 {
			return 42
		}
		return 32
	}
	percent := 32 + int((loaded*50)/total)
	if percent < 32 {
		return 32
	}
	if percent > 82 {
		return 82
	}
	return percent
}
