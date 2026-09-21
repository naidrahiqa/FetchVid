package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type Settings struct {
	OutputDir   string `json:"outputDir"`
	CookiesFile string `json:"cookiesFile"`
	Concurrent  int    `json:"concurrent"`
	Theme       string `json:"theme"`
}

func userHome() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

func DefaultSettings() *Settings {
	return &Settings{
		OutputDir:   filepath.Join(userHome(), "Downloads", "FetchVid"),
		Concurrent:  3,
		Theme:       "dark",
	}
}

func settingsPath() string {
	var dir string
	if runtime.GOOS == "windows" {
		dir = filepath.Join(os.Getenv("APPDATA"), "FetchVid")
	} else {
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" {
			xdg = filepath.Join(userHome(), ".config")
		}
		dir = filepath.Join(xdg, "FetchVid")
	}
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
