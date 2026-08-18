package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

var dailyCache struct {
	sync.Mutex
	date string
	url  string
}

// fetchBingDaily 获取必应每日壁纸 URL
func fetchBingDaily() string {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get("https://cn.bing.com/HPImageArchive.aspx?format=js&idx=0&n=1&mkt=zh-CN")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var data struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	if len(data.Images) == 0 || data.Images[0].URL == "" {
		return ""
	}
	return "https://cn.bing.com" + data.Images[0].URL
}

func getDailyImage() string {
	today := time.Now().Format("2006-01-02")
	dailyCache.Lock()
	defer dailyCache.Unlock()

	if dailyCache.date == today && dailyCache.url != "" {
		return dailyCache.url
	}

	url := fetchBingDaily()
	if url == "" {
		return ""
	}
	dailyCache.date = today
	dailyCache.url = url
	return url
}

func dailyImageHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"url": getDailyImage()})
}
