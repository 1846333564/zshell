package updatesvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"zshell/backend/internal/appinfo"
)

type Service struct {
	client *http.Client
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
}

func NewService() *Service {
	return &Service{
		client: &http.Client{Timeout: 45 * time.Second},
	}
}

func (s *Service) Check(ctx context.Context) (CheckResult, error) {
	release, err := s.latestRelease(ctx)
	if err != nil {
		return CheckResult{}, err
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
	release, err := s.latestRelease(ctx)
	if err != nil {
		return ApplyResult{}, err
	}

	latestVersion := normalizeVersion(release.TagName)
	if compareVersions(latestVersion, appinfo.Version) <= 0 {
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

	downloadPath, digest, err := s.downloadExecutable(ctx, asset)
	if err != nil {
		return ApplyResult{}, err
	}

	expectedDigest := assetDigest(asset)
	if expectedDigest == "" {
		expectedDigest, err = s.findSHA256Digest(ctx, release.Assets, asset.Name)
		if err != nil {
			_ = os.Remove(downloadPath)
			return ApplyResult{}, err
		}
	}
	if expectedDigest != "" && !strings.EqualFold(digest, expectedDigest) {
		_ = os.Remove(downloadPath)
		return ApplyResult{}, fmt.Errorf("downloaded update checksum mismatch")
	}

	if err := scheduleReplacement(downloadPath); err != nil {
		_ = os.Remove(downloadPath)
		return ApplyResult{}, err
	}

	go func() {
		time.Sleep(600 * time.Millisecond)
		os.Exit(0)
	}()

	return ApplyResult{
		OK:             true,
		CurrentVersion: appinfo.Version,
		LatestVersion:  latestVersion,
		Message:        "更新已下载，应用即将重启",
	}, nil
}

func (s *Service) latestRelease(ctx context.Context) (githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", appinfo.GitHubOwner, appinfo.GitHubRepo)
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
		return githubRelease{}, fmt.Errorf("github release not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return githubRelease{}, fmt.Errorf("github release request failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("decode github release: %w", err)
	}
	return release, nil
}

func (s *Service) downloadExecutable(ctx context.Context, asset githubAsset) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", appinfo.ProductName+"/"+appinfo.Version)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("download update failed: %s", resp.Status)
	}

	dir := filepath.Join(os.TempDir(), "zshell-update")
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
	if _, err := io.Copy(io.MultiWriter(file, hash), resp.Body); err != nil {
		_ = os.Remove(target)
		return "", "", fmt.Errorf("write update temp file: %w", err)
	}

	return target, hex.EncodeToString(hash.Sum(nil)), nil
}

func scheduleReplacement(sourcePath string) error {
	targetPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}

	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("zshell-apply-update-%d.ps1", time.Now().UnixNano()))
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$source = '%s'
$target = '%s'
$pidToWait = %d
try {
  Wait-Process -Id $pidToWait -Timeout 60 -ErrorAction SilentlyContinue
} catch {}
Start-Sleep -Milliseconds 800
Copy-Item -LiteralPath $source -Destination $target -Force
Start-Process -FilePath $target
Remove-Item -LiteralPath $source -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $MyInvocation.MyCommand.Path -Force -ErrorAction SilentlyContinue
`, powerShellQuote(sourcePath), powerShellQuote(targetPath), os.Getpid())

	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return fmt.Errorf("write update helper: %w", err)
	}

	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update helper: %w", err)
	}
	return nil
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.ToLower(value), "v")
	return value
}

func compareVersions(left string, right string) int {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	for i := 0; i < len(leftParts) || i < len(rightParts); i++ {
		leftValue := 0
		rightValue := 0
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return 0
}

func versionParts(value string) []int {
	value = normalizeVersion(value)
	rawParts := strings.Split(value, ".")
	parts := make([]int, 0, len(rawParts))
	for _, raw := range rawParts {
		number, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || number < 0 {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, number)
	}
	return parts
}

func findExecutableAsset(assets []githubAsset, version string) githubAsset {
	exactName := strings.ToLower(appinfo.ReleaseAssetName(version))
	for _, asset := range assets {
		if strings.ToLower(asset.Name) == exactName {
			return asset
		}
	}
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.HasPrefix(name, "zshell.") && strings.HasSuffix(name, ".exe") {
			return asset
		}
	}
	return githubAsset{}
}

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

func (s *Service) findSHA256Digest(ctx context.Context, assets []githubAsset, exeName string) (string, error) {
	candidates := []string{exeName + ".sha256", "sha256.txt"}
	for _, candidate := range candidates {
		for _, asset := range assets {
			if strings.EqualFold(asset.Name, candidate) && asset.BrowserDownloadURL != "" {
				digest, err := s.downloadSHA256(ctx, asset.BrowserDownloadURL, exeName)
				if err != nil {
					return "", err
				}
				return digest, nil
			}
		}
	}
	return "", nil
}

func (s *Service) downloadSHA256(ctx context.Context, url string, exeName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", appinfo.ProductName+"/"+appinfo.Version)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download checksum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download checksum failed: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
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

func powerShellQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
