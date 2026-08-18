package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var db *sql.DB

type Dial struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Image   string `json:"image"`
	GroupID int64  `json:"group_id"`
	Sort    int    `json:"sort"`
}

type Group struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parent_id"`
	Name     string `json:"name"`
	Sort     int    `json:"sort"`
}

type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func dataDir() string {
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		dir = "data"
	}
	return dir
}

func initDB() {
	var err error
	dir := dataDir()
	if err = os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}
	if err = os.MkdirAll(filepath.Join(dir, "uploads"), 0o755); err != nil {
		log.Fatalf("创建上传目录失败: %v", err)
	}

	dbPath := filepath.Join(dir, "speeddials.db")
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	// 启用 WAL 模式提升并发读写性能
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA foreign_keys=ON;")

	schema := `
CREATE TABLE IF NOT EXISTS dials (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	title      TEXT NOT NULL,
	url        TEXT NOT NULL,
	image      TEXT NOT NULL DEFAULT '',
	group_id   INTEGER NOT NULL DEFAULT 0,
	sort       INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS groups (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	parent_id INTEGER NOT NULL DEFAULT 0,
	name      TEXT NOT NULL,
	sort      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL DEFAULT ''
);`

	if _, err = db.Exec(schema); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 兼容旧库：给 dials 补充 group_id 列
	migrate()
}

func migrate() {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('dials') WHERE name = 'group_id'").Scan(&count)
	if err != nil {
		return
	}
	if count == 0 {
		db.Exec("ALTER TABLE dials ADD COLUMN group_id INTEGER NOT NULL DEFAULT 0")
	}
}

func initSettings() {
	defaults := map[string]string{
		"site_title":    "我的导航",
		"site_subtitle": "",
		"columns":       "6",
		"daily_image":   "0",
		"footer_html":   "",
	}
	for k, v := range defaults {
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM settings WHERE key = ?", k).Scan(&exists)
		if err != nil {
			log.Fatalf("读取设置失败: %v", err)
		}
		if exists == 0 {
			if _, err := db.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", k, v); err != nil {
				log.Fatalf("写入默认设置失败: %v", err)
			}
		}
	}

	// 管理员用户名：环境变量 ADMIN_USER 优先，其次数据库已有值，最后默认 admin
	user := os.Getenv("ADMIN_USER")
	if user == "" {
		_ = db.QueryRow("SELECT value FROM settings WHERE key = 'username'").Scan(&user)
	}
	if user == "" {
		user = "admin"
	}
	if _, err := db.Exec(
		"INSERT INTO settings (key, value) VALUES ('username', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		user); err != nil {
		log.Fatalf("写入用户名失败: %v", err)
	}

	// 管理员密码：环境变量 ADMIN_PASSWORD 优先，其次数据库已有值，最后默认 123456
	pw := os.Getenv("ADMIN_PASSWORD")
	if pw == "" {
		var hash string
		_ = db.QueryRow("SELECT value FROM settings WHERE key = 'password_hash'").Scan(&hash)
		if hash == "" {
			pw = "123456"
		}
	}
	if pw != "" {
		h, herr := hashPassword(pw)
		if herr != nil {
			log.Fatalf("生成密码失败: %v", herr)
		}
		if _, err := db.Exec(
			"INSERT INTO settings (key, value) VALUES ('password_hash', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
			h); err != nil {
			log.Fatalf("写入密码失败: %v", err)
		}
	}
}

func getAdminUser() string {
	var user string
	_ = db.QueryRow("SELECT value FROM settings WHERE key = 'username'").Scan(&user)
	if user == "" {
		return "admin"
	}
	return user
}
