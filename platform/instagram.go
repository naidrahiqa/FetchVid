package platform

import (
	"os/exec"
	"strings"
)

type Instagram struct{}

func (i *Instagram) Name() string { return "instagram" }

func (i *Instagram) Match(rawurl string) bool {
	u := strings.ToLower(rawurl)
	return contains(u, "instagram.com") || contains(u, "instagr.am")
}

func (i *Instagram) ExtractURLs(rawurl string, cookies string) ([]VideoInfo, error) {
	// Use yt-dlp --flat-playlist to extract URLs from Instagram profile
	args := []string{
		"--flat-playlist", "--dump-json",
		"--no-warnings", "--ignore-errors",
		"--no-check-certificates", "--geo-bypass",
	}
	if cookies != "" {
		args = append(args, "--cookies", cookies)
	}
	args = append(args, rawurl)

	cmd := exec.Command("yt-dlp", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []VideoInfo
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry, err := parseVideoEntry(line, "instagram")
		if err != nil {
			continue
		}
		if !seen[entry.URL] {
			seen[entry.URL] = true
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (i *Instagram) ConsoleScripts() []ScriptInfo {
	return []ScriptInfo{
		{
			Platform: "instagram",
			Label:    "Instagram - Reels",
			Script:   "copy([...document.querySelectorAll('a[href*=\"/reel/\"], a[href*=\"/p/\"]')].map(a=>a.href.split(\"?\")[0].split(\"/\").slice(0,7).join(\"/\")+\"/\").filter((v,i,a)=>a.indexOf(v)===i).join('\\n'))",
			Desc:     "Semua reels & posts dari profile Instagram",
		},
	}
}
