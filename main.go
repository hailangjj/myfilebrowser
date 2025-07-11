package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	rice "github.com/GeertJohan/go.rice"
)

// FileItem 代表一个文件或目录项。
type FileItem struct {
	Name     string
	Path     string
	IsDir    bool
	IsMedia  bool
	MimeType string
}

// DirectoryView 用于存储目录视图的信息。
type DirectoryView struct {
	CurrentPath string
	Breadcrumbs []Breadcrumb
	Folders     []FileItem
	Files       []FileItem
}

// Breadcrumb 代表面包屑导航中的一个部分。
type Breadcrumb struct {
	Name string
	Path string
}

// rootDir 用于存储命令行指定的根目录路径。
var rootDir string

// tmpl 用于存储解析后的HTML模板。
var tmpl *template.Template

func main() {
	addr := flag.String("addr", "0.0.0.0", "Listen address")
	port := flag.String("port", "8080", "Listen port")
	flag.StringVar(&rootDir, "dir", "./srv", "Root directory to serve")
	flag.Parse()

	// 加载模板
	templateBox, err := rice.FindBox("templates")
	if err != nil {
		log.Fatal(err)
	}

	tmpl = template.New("").Funcs(template.FuncMap{
		"hasPrefix":     strings.HasPrefix,
		"isPreviewable": isPreviewable,
	})

	files := []string{"index.tmpl", "preview.tmpl"}
	for _, name := range files {
		// 为避免变量 err 遮蔽第 54 行的声明，使用新的变量名 errTemplate
		content, errTemplate := templateBox.String(name)
		if errTemplate != nil {
			log.Fatalf("Failed to load template %s: %v", name, err)
		}
		_, err = tmpl.New(name).Parse(content)
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", name, err)
		}
	}

	// 加载静态文件
	staticFileBox, err := rice.FindBox("static")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(staticFileBox.HTTPBox())))

	http.HandleFunc("/preview", previewHandler)
	http.HandleFunc("/", fileHandler)

	log.Printf("Serving %s on http://%s:%s\n", rootDir, *addr, *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", *addr, *port), nil))
}

// fileHandler 处理文件请求。
func fileHandler(w http.ResponseWriter, r *http.Request) {
	reqPath := filepath.Clean(r.URL.Path)
	absPath := filepath.Join(rootDir, reqPath)

	info, err := os.Stat(absPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	rootAbs, _ := filepath.EvalSymlinks(rootDir)
	absPathEval, _ := filepath.EvalSymlinks(absPath)
	if rootAbs != "" && !strings.HasPrefix(absPathEval, rootAbs) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if !info.IsDir() {
		serveFile(w, r, absPath, info.Name())
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		http.Error(w, "Cannot read directory", http.StatusInternalServerError)
		return
	}

	var folders []FileItem
	var files []FileItem

	for _, entry := range entries {
		itemName := entry.Name()
		itemPath := filepath.ToSlash(filepath.Join(reqPath, itemName))
		mimeType := "application/octet-stream"
		isMedia := false
		if !entry.IsDir() {
			ext := filepath.Ext(itemName)
			mimeType = mime.TypeByExtension(ext)
			isMedia = strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") || strings.HasPrefix(mimeType, "audio/")
		}
		item := FileItem{
			Name:     itemName,
			Path:     itemPath,
			IsDir:    entry.IsDir(),
			IsMedia:  isMedia,
			MimeType: mimeType,
		}
		if entry.IsDir() {
			folders = append(folders, item)
		} else {
			files = append(files, item)
		}
	}

	sort.SliceStable(folders, func(i, j int) bool { return folders[i].Name < folders[j].Name })
	sort.SliceStable(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	breadcrumbs := buildBreadcrumbs(reqPath)

	curDir := filepath.Base(reqPath)
	if curDir == "." || curDir == string(filepath.Separator) || curDir == "" {
		curDir = "Home"
	}

	view := DirectoryView{
		CurrentPath: filepath.Base(reqPath),
		Breadcrumbs: breadcrumbs,
		Folders:     folders,
		Files:       files,
	}

	if err := tmpl.ExecuteTemplate(w, "index.tmpl", view); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// buildBreadcrumbs 生成面包屑导航。
func buildBreadcrumbs(path string) []Breadcrumb {
	var crumbs []Breadcrumb

	// 统一使用 /，去除开头的 / 保证是相对路径
	cleanPath := strings.TrimPrefix(filepath.ToSlash(path), "/")

	parts := strings.Split(cleanPath, "/")

	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if current == "" {
			current = part
		} else {
			current = current + "/" + part
		}
		crumbs = append(crumbs, Breadcrumb{
			Name: part,
			Path: "/" + current, // 保证每个 Path 都以 / 开头，方便跳转
		})
	}

	return crumbs
}

// isPreviewable 判断文件类型是否可预览。
func isPreviewable(mime string) bool {
	return strings.HasPrefix(mime, "text/") ||
		strings.HasPrefix(mime, "image/") ||
		strings.HasPrefix(mime, "audio/") ||
		strings.HasPrefix(mime, "video/") ||
		mime == "application/pdf"
}

func serveFile(w http.ResponseWriter, r *http.Request, absPath, fileName string) {
	ext := filepath.Ext(fileName)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mimeType)

	if r.URL.Query().Get("download") == "1" {
		// 强制下载
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	} else {
		// 正常打开
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", fileName))
	}

	http.ServeFile(w, r, absPath)
}

func previewHandler(w http.ResponseWriter, r *http.Request) {
	queryPath := r.URL.Query().Get("path")
	if queryPath == "" {
		http.Error(w, "Missing path", http.StatusBadRequest)
		return
	}

	absPath := filepath.Join(rootDir, filepath.Clean(queryPath))
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// 校验路径合法性
	rootAbs, _ := filepath.EvalSymlinks(rootDir)
	absPathEval, _ := filepath.EvalSymlinks(absPath)
	if rootAbs != "" && !strings.HasPrefix(absPathEval, rootAbs) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	ext := filepath.Ext(info.Name())
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// 渲染预览页模板
	err = tmpl.ExecuteTemplate(w, "preview.tmpl", map[string]interface{}{
		"Name":     info.Name(),
		"Path":     queryPath,
		"MimeType": mimeType,
		"Size":     info.Size(),
	})
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
