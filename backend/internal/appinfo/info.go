package appinfo

var Version = "0.3.18"

const (
	ProductName     = "wiShell"
	Company         = "重庆创翼科技有限公司"
	Developer       = "zly"
	Channel         = "暂时内测版"
	GitHubOwner     = "1846333564"
	GitHubRepo      = "zshell"
	ReleaseAssetTpl = "wiShell.%s.exe"
)

type Info struct {
	ProductName string `json:"productName"`
	Version     string `json:"version"`
	Company     string `json:"company"`
	Developer   string `json:"developer"`
	Channel     string `json:"channel"`
	Repository  string `json:"repository"`
}

func Current() Info {
	return Info{
		ProductName: ProductName,
		Version:     Version,
		Company:     Company,
		Developer:   Developer,
		Channel:     Channel,
		Repository:  "https://github.com/" + GitHubOwner + "/" + GitHubRepo,
	}
}

func ReleaseAssetName(version string) string {
	return "wiShell." + version + ".exe"
}
