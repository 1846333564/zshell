package updatesvc

import (
	"context"
	"fmt"
	"time"

	"zshell/backend/internal/appinfo"
)

func reportProgress(report ProgressReporter, event ProgressEvent) {
	if report == nil {
		return
	}
	if event.Percent < 0 {
		event.Percent = 0
	}
	if event.Percent > 100 {
		event.Percent = 100
	}
	report(event)
}

func waitBeforeRetry(ctx context.Context, attempt int) {
	if attempt >= updateDownloadAttempts {
		return
	}
	timer := time.NewTimer(time.Duration(attempt) * 1200 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func explainCheckError(err error) error {
	if err == nil {
		return nil
	}
	if IsStopped(err) {
		return err
	}
	return fmt.Errorf("检查 GitHub Release 失败。GitHub API 可能被限流，或当前网络无法访问 GitHub。请稍后重试，或手动打开 %s 下载最新版本。原始错误：%w", manualReleaseURL(), err)
}

func explainDownloadError(action string, downloadURL string, err error) error {
	if err == nil {
		return nil
	}
	if IsStopped(err) {
		return err
	}
	return fmt.Errorf("%s：无法稳定连接 GitHub Release 下载地址。请检查网络或代理后重试，或手动下载 %s。原始错误：%w", action, downloadURL, err)
}

func manualReleaseURL() string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/latest", appinfo.GitHubOwner, appinfo.GitHubRepo)
}
