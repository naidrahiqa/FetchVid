package platform

import "encoding/json"

// parseVideoEntry parses a yt-dlp JSON line into a VideoInfo
func parseVideoEntry(line, source string) (VideoInfo, error) {
	var raw struct {
		WebpageURL string `json:"webpage_url"`
		URL        string `json:"url"`
		Title      string `json:"title"`
		ID         string `json:"id"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return VideoInfo{}, err
	}
	u := raw.WebpageURL
	if u == "" {
		u = raw.URL
	}
	if u == "" {
		return VideoInfo{}, nil
	}
	return VideoInfo{
		URL:    u,
		Title:  raw.Title,
		Source: source,
	}, nil
}
