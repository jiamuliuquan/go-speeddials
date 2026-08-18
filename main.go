package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static
var staticFS embed.FS

func main() {
	initDB()
	initSettings()

	mux := http.NewServeMux()

	// 静态资源
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("加载静态资源失败: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	// 上传的图片
	mux.Handle("/uploads/", http.StripPrefix("/uploads/",
		http.FileServer(http.Dir(dataDir()+"/uploads"))))

	// API 路由
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		login(w, r)
	})
	mux.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		logout(w, r)
	})
	mux.HandleFunc("/api/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		authStatus(w, r)
	})

	mux.HandleFunc("/api/dials", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listDials(w, r)
		case http.MethodPost:
			if !requireAuth(w, r) {
				return
			}
			createDial(w, r)
		default:
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
		}
	})
	mux.HandleFunc("/api/dials/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r) {
				return
			}
			updateDial(w, r)
		case http.MethodDelete:
			if !requireAuth(w, r) {
				return
			}
			deleteDial(w, r)
		default:
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
		}
	})

	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listGroups(w, r)
		case http.MethodPost:
			if !requireAuth(w, r) {
				return
			}
			createGroup(w, r)
		default:
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
		}
	})
	mux.HandleFunc("/api/groups/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r) {
				return
			}
			updateGroup(w, r)
		case http.MethodDelete:
			if !requireAuth(w, r) {
				return
			}
			deleteGroup(w, r)
		default:
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
		}
	})

	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getSettings(w, r)
		case http.MethodPut:
			if !requireAuth(w, r) {
				return
			}
			updateSettings(w, r)
		default:
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
		}
	})

	mux.HandleFunc("/api/reorder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		if !requireAuth(w, r) {
			return
		}
		reorder(w, r)
	})

	mux.HandleFunc("/api/daily", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		dailyImageHandler(w, r)
	})

	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		if !requireAuth(w, r) {
			return
		}
		uploadImage(w, r)
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("快速拨号已启动，监听 %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
