package platform

type ScriptInfo struct {
	Platform string `json:"platform"`
	Label    string `json:"label"`
	Script   string `json:"script"`
	Desc     string `json:"desc"`
}

type Platform interface {
	Name() string
	Match(url string) bool
	ExtractURLs(url string, cookies string) ([]VideoInfo, error)
	ConsoleScripts() []ScriptInfo
}

var allPlatforms = []Platform{
	&Facebook{},
	&Instagram{},
	&TikTok{},
}

func All() []Platform {
	return allPlatforms
}

func Detect(url string) Platform {
	for _, p := range allPlatforms {
		if p.Match(url) {
			return p
		}
	}
	return nil
}
