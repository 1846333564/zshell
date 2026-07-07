package updatesvc

import (
	"strconv"
	"strings"

	"wiShell/backend/internal/appinfo"
)

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
		if strings.HasPrefix(name, "wiShell.") && strings.HasSuffix(name, ".exe") {
			return asset
		}
	}
	return githubAsset{}
}
