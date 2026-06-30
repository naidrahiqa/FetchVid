package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	OutputDir   string `json:"outputDir"`
	CookiesFile string `json:"cookiesFile"`
	Concurrent  int    `json:"concurrent"`
	Theme       string `json:"theme"`
}

func DefaultSettings() *Settings {
	return &Settings{
		OutputDir:   filepath.Join(os.Getenv("USERPROFILE"), "Downloads", "FetchVid"),
		Concurrent:  3,
		Theme:       "dark",
	}
}

func settingsPath() string {
	dir := filepath.Join(os.Getenv("APPDATA"), "FetchVid")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.json")
}

func LoadSettings() *Settings {
	s := DefaultSettings()
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return s
	}
	json.Unmarshal(data, s)
	if s.Concurrent < 1 {
		s.Concurrent = 3
	}
	return s
}

func SaveSettings(s *Settings) {
	data, _ := json.MarshalIndent(s, "", "  ")
	os.WriteFile(settingsPath(), data, 0644)
}
