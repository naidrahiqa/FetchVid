package app

import (
	"sync"
)

type Downloader struct {
	mu      sync.Mutex
	running bool
	paused  bool
}

func NewDownloader() *Downloader {
	return &Downloader{}
}

func (d *Downloader) Start(urls []VideoInfo, concurrent int, outputDir, cookiesFile string, progressFn func(Progress)) {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return
	}
	d.running = true
	d.mu.Unlock()

	go d.run(urls, concurrent, outputDir, cookiesFile, progressFn)
}

func (d *Downloader) run(urls []VideoInfo, concurrent int, outputDir, cookiesFile string, progressFn func(Progress)) {
	defer func() {
		d.mu.Lock()
		d.running = false
		d.mu.Unlock()
	}()

	total := len(urls)
	progress := Progress{
		Total:     total,
		Completed: 0,
		Success:   0,
		Failed:    0,
	}

	for i, v := range urls {
		d.mu.Lock()
		if !d.running {
			d.mu.Unlock()
			return
		}
		for d.paused {
			d.mu.Unlock()
			// wait briefly then recheck
			d.mu.Lock()
		}
		d.mu.Unlock()

		progress.CurrentURL = v.URL
		progress.Completed = i + 1
		progress.Success = i + 1 // simplified: assume all succeed for stub
		if progressFn != nil {
			progressFn(progress)
		}
	}

	progress.Completed = total
	if progressFn != nil {
		progressFn(progress)
	}
}

func (d *Downloader) Pause() {
	d.mu.Lock()
	d.paused = true
	d.mu.Unlock()
}

func (d *Downloader) Resume() {
	d.mu.Lock()
	d.paused = false
	d.mu.Unlock()
}

func (d *Downloader) Stop() {
	d.mu.Lock()
	d.running = false
	d.mu.Unlock()
}

type Progress struct {
	Total      int     `json:"total"`
	Completed  int     `json:"completed"`
	Success    int     `json:"success"`
	Failed     int     `json:"failed"`
	Percent    float64 `json:"percent"`
	CurrentURL string  `json:"currentUrl"`
}
