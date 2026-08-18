package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseID(r *http.Request) (int64, error) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("无效的 ID")
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的 ID")
	}
	return id, nil
}

// ---------- 拨号项 ----------

func listDials(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, url, image, group_id, sort FROM dials ORDER BY sort ASC, id ASC")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "查询失败")
		return
	}
	defer rows.Close()

	dials := []Dial{}
	for rows.Next() {
		var d Dial
		if err := rows.Scan(&d.ID, &d.Title, &d.URL, &d.Image, &d.GroupID, &d.Sort); err != nil {
			writeErr(w, http.StatusInternalServerError, "读取失败")
			return
		}
		dials = append(dials, d)
	}
	writeJSON(w, http.StatusOK, dials)
}

func createDial(w http.ResponseWriter, r *http.Request) {
	var d Dial
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if strings.TrimSpace(d.Title) == "" || strings.TrimSpace(d.URL) == "" {
		writeErr(w, http.StatusBadRequest, "标题和链接不能为空")
		return
	}
	if !strings.HasPrefix(d.URL, "http://") && !strings.HasPrefix(d.URL, "https://") {
		d.URL = "https://" + d.URL
	}
	res, err := db.Exec("INSERT INTO dials (title, url, image, group_id, sort) VALUES (?, ?, ?, ?, ?)",
		d.Title, d.URL, d.Image, d.GroupID, d.Sort)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "保存失败")
		return
	}
	d.ID, _ = res.LastInsertId()
	writeJSON(w, http.StatusCreated, d)
}

func updateDial(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	var d Dial
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if strings.TrimSpace(d.Title) == "" || strings.TrimSpace(d.URL) == "" {
		writeErr(w, http.StatusBadRequest, "标题和链接不能为空")
		return
	}
	if !strings.HasPrefix(d.URL, "http://") && !strings.HasPrefix(d.URL, "https://") {
		d.URL = "https://" + d.URL
	}
	res, err := db.Exec("UPDATE dials SET title = ?, url = ?, image = ?, group_id = ?, sort = ? WHERE id = ?",
		d.Title, d.URL, d.Image, d.GroupID, d.Sort, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "保存失败")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeErr(w, http.StatusNotFound, "条目不存在")
		return
	}
	d.ID = id
	writeJSON(w, http.StatusOK, d)
}

func deleteDial(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := db.Exec("DELETE FROM dials WHERE id = ?", id); err != nil {
		writeErr(w, http.StatusInternalServerError, "删除失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---------- 目录 ----------

func listGroups(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, parent_id, name, sort FROM groups ORDER BY sort ASC, id ASC")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "查询失败")
		return
	}
	defer rows.Close()

	groups := []Group{}
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.ParentID, &g.Name, &g.Sort); err != nil {
			writeErr(w, http.StatusInternalServerError, "读取失败")
			return
		}
		groups = append(groups, g)
	}
	writeJSON(w, http.StatusOK, groups)
}

func createGroup(w http.ResponseWriter, r *http.Request) {
	var g Group
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if strings.TrimSpace(g.Name) == "" {
		writeErr(w, http.StatusBadRequest, "目录名不能为空")
		return
	}
	res, err := db.Exec("INSERT INTO groups (parent_id, name, sort) VALUES (?, ?, ?)",
		g.ParentID, g.Name, g.Sort)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "保存失败")
		return
	}
	g.ID, _ = res.LastInsertId()
	writeJSON(w, http.StatusCreated, g)
}

func updateGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	var g Group
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if strings.TrimSpace(g.Name) == "" {
		writeErr(w, http.StatusBadRequest, "目录名不能为空")
		return
	}
	if g.ParentID == id {
		writeErr(w, http.StatusBadRequest, "不能移动到自身")
		return
	}
	if isDescendant(g.ParentID, id) {
		writeErr(w, http.StatusBadRequest, "不能移动到自己的子目录")
		return
	}
	res, err := db.Exec("UPDATE groups SET parent_id = ?, name = ?, sort = ? WHERE id = ?",
		g.ParentID, g.Name, g.Sort, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "保存失败")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeErr(w, http.StatusNotFound, "目录不存在")
		return
	}
	g.ID = id
	writeJSON(w, http.StatusOK, g)
}

// isDescendant 判断 candidate 是否是 ancestor 的后代目录
func isDescendant(candidate, ancestor int64) bool {
	cur := candidate
	for cur != 0 {
		if cur == ancestor {
			return true
		}
		var parent int64
		if err := db.QueryRow("SELECT parent_id FROM groups WHERE id = ?", cur).Scan(&parent); err != nil {
			return false
		}
		cur = parent
	}
	return false
}

func deleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := deleteGroupTree(id); err != nil {
		writeErr(w, http.StatusInternalServerError, "删除失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// deleteGroupTree 递归删除目录及其所有子目录、子目录下的拨号项
func deleteGroupTree(id int64) error {
	// 收集所有后代目录 ID
	var ids []int64
	queue := []int64{id}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		ids = append(ids, cur)

		rows, err := db.Query("SELECT id FROM groups WHERE parent_id = ?", cur)
		if err != nil {
			return err
		}
		var children []int64
		for rows.Next() {
			var cid int64
			if err := rows.Scan(&cid); err != nil {
				rows.Close()
				return err
			}
			children = append(children, cid)
		}
		rows.Close()
		queue = append(queue, children...)
	}

	for _, gid := range ids {
		if _, err := db.Exec("DELETE FROM dials WHERE group_id = ?", gid); err != nil {
			return err
		}
	}
	for i := len(ids) - 1; i >= 0; i-- {
		if _, err := db.Exec("DELETE FROM groups WHERE id = ?", ids[i]); err != nil {
			return err
		}
	}
	return nil
}

// ---------- 排序 ----------

func reorder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type string  `json:"type"`
		IDs  []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if body.Type != "dial" && body.Type != "group" {
		writeErr(w, http.StatusBadRequest, "无效的类型")
		return
	}
	table := "dials"
	if body.Type == "group" {
		table = "groups"
	}
	for i, id := range body.IDs {
		if _, err := db.Exec("UPDATE "+table+" SET sort = ? WHERE id = ?", i, id); err != nil {
			writeErr(w, http.StatusInternalServerError, "保存排序失败")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---------- 认证 ----------

func login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if body.Username != getAdminUser() || !checkPassword(body.Password) {
		writeErr(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}
	setSession(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func logout(w http.ResponseWriter, r *http.Request) {
	clearSession(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func authStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": isAuthenticated(r)})
}

// ---------- 站点设置 ----------

func getSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT key, value FROM settings")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "查询失败")
		return
	}
	defer rows.Close()

	settings := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			writeErr(w, http.StatusInternalServerError, "读取失败")
			return
		}
		if k == "password_hash" || k == "username" {
			continue
		}
		settings[k] = v
	}
	writeJSON(w, http.StatusOK, settings)
}

func updateSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeErr(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	for k, v := range settings {
		if k == "password_hash" || k == "username" {
			continue
		}
		if _, err := db.Exec(
			"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
			k, v); err != nil {
			writeErr(w, http.StatusInternalServerError, "保存失败")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---------- 图片上传 ----------

func uploadImage(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(16 << 20) // 最大 16MB
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "未收到文件")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".webp" && ext != ".svg" && ext != ".ico" {
		writeErr(w, http.StatusBadRequest, "仅支持 png/jpg/jpeg/gif/webp/svg/ico 格式")
		return
	}

	buf := make([]byte, 8)
	rand.Read(buf)
	name := hex.EncodeToString(buf) + ext
	dst := filepath.Join(dataDir(), "uploads", name)

	out, err := os.Create(dst)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "保存文件失败")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		writeErr(w, http.StatusInternalServerError, "保存文件失败")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"url": "/uploads/" + name})
}
