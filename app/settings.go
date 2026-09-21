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
		OutputDir:  filepath.Join(userHome(), "Downloads", "FetchVid"),
		Concurrent: 3,
		Theme:      "dark",
	}
}

func settingsDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "FetchVid")
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(userHome(), ".config")
	}
	return filepath.Join(xdg, "FetchVid")
}

func LoadSettings() *Settings {
	dir := settingsDir()
	os.MkdirAll(dir, 0755)

	s := DefaultSettings()
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, s); err != nil {
		return s
	}
	if s.Concurrent < 1 {
		s.Concurrent = 3
	}
	if s.Concurrent > 10 {
		s.Concurrent = 10
	}
	return s
}

func SaveSettings(s *Settings) {
	dir := settingsDir()
	os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}
