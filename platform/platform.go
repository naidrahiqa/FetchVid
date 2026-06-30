package platform

type VideoInfo struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Source string `json:"source"`
}

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

// All returns all registered platform extractors
func All() []Platform {
	return []Platform{
		&Facebook{},
		&Instagram{},
		&TikTok{},
	}
}

// Detect detects which platform a URL belongs to
func Detect(url string) Platform {
	for _, p := range All() {
		if p.Match(url) {
			return p
		}
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
