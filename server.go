package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
)

//go:embed web/*
var webFS embed.FS

type ProgressEvent struct {
	Stage   string `json:"stage"`             // "fetching", "downloading", "converting", "completed", "error"
	Title   string `json:"title,omitempty"`
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Message string `json:"message,omitempty"`
	Error   bool   `json:"error,omitempty"`
}

type DownloadRequest struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Threads    int    `json:"threads"`
	ChunkSize  int    `json:"chunkSize"`
	KeepFolder bool   `json:"keepFolder"`
}

type BookInfo struct {
	Name          string `json:"name"`
	SizeFormatted string `json:"sizeFormatted"`
	Modified      string `json:"modified"`
}

var (
	sseClientsMu  sync.Mutex
	sseClients    = make(map[chan ProgressEvent]bool)
	isDownloading sync.Mutex
)

func broadcastProgress(event ProgressEvent) {
	sseClientsMu.Lock()
	defer sseClientsMu.Unlock()
	for clientChan := range sseClients {
		select {
		case clientChan <- event:
		default:
		}
	}
}

func startUI() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Gagal membuka port server lokal: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	appURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	mux := http.NewServeMux()

	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Gagal membaca asset web: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	mux.HandleFunc("/api/progress", handleProgressSSE)
	mux.HandleFunc("/api/download", handleDownloadAPI)
	mux.HandleFunc("/api/books", handleBooksAPI)
	mux.HandleFunc("/api/open-file", handleOpenFileAPI)
	mux.HandleFunc("/api/open-folder", handleOpenFolderAPI)

	fmt.Printf("====================================================\n")
	fmt.Printf("  AnyFlip Downloader - Windowed UI Aktif\n")
	fmt.Printf("  URL: %s\n", appURL)
	fmt.Printf("====================================================\n")

	go func() {
		time.Sleep(250 * time.Millisecond)
		launchAppWindow(appURL)
	}()

	server := &http.Server{Handler: mux}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Printf("Server berhenti: %v", err)
	}
}

func launchAppWindow(appURL string) {
	edgePaths := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	}
	for _, p := range edgePaths {
		if _, err := os.Stat(p); err == nil {
			cmd := exec.Command(p, fmt.Sprintf("--app=%s", appURL), "--window-size=920,740")
			if err := cmd.Start(); err == nil {
				return
			}
		}
	}

	chromePaths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	}
	for _, p := range chromePaths {
		if _, err := os.Stat(p); err == nil {
			cmd := exec.Command(p, fmt.Sprintf("--app=%s", appURL), "--window-size=920,740")
			if err := cmd.Start(); err == nil {
				return
			}
		}
	}

	exec.Command("cmd", "/c", "start", appURL).Start()
}

func handleProgressSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan ProgressEvent, 50)
	sseClientsMu.Lock()
	sseClients[clientChan] = true
	sseClientsMu.Unlock()

	defer func() {
		sseClientsMu.Lock()
		delete(sseClients, clientChan)
		sseClientsMu.Unlock()
	}()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case event := <-clientChan:
			data, err := json.Marshal(event)
			if err == nil {
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}

func handleDownloadAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		http.Error(w, "URL harus diisi", http.StatusBadRequest)
		return
	}

	if req.Threads <= 0 {
		req.Threads = 4
	}
	if req.ChunkSize <= 0 {
		req.ChunkSize = 10
	}

	if !isDownloading.TryLock() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "Proses unduhan lain sedang berjalan"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})

	go func() {
		defer isDownloading.Unlock()
		runUIDownload(req)
	}()
}

func runUIDownload(req DownloadRequest) {
	broadcastProgress(ProgressEvent{
		Stage:   "fetching",
		Message: "Menghubungi server AnyFlip...",
	})

	anyflipURL, err := url.Parse(req.URL)
	if err != nil {
		broadcastProgress(ProgressEvent{
			Stage:   "error",
			Error:   true,
			Message: "Format URL tidak valid: " + err.Error(),
		})
		return
	}

	sanitizeURL(anyflipURL)

	configjs, err := downloadConfigJSFile(anyflipURL)
	if err != nil {
		broadcastProgress(ProgressEvent{
			Stage:   "error",
			Error:   true,
			Message: "Gagal mengunduh config.js dari AnyFlip: " + err.Error(),
		})
		return
	}

	bookTitle := req.Title
	if bookTitle == "" {
		bookTitle, err = getBookTitle(configjs)
		if err != nil {
			bookTitle = path.Base(anyflipURL.String())
		}
	}

	safeTitle := govalidator.SafeFileName(bookTitle)
	if safeTitle == "" {
		safeTitle = path.Base(anyflipURL.Path)
	}
	if safeTitle == "" || safeTitle == "." {
		safeTitle = "anyflip-download"
	}
	safeTitle = strings.Trim(strings.TrimSpace(safeTitle), ".")
	if safeTitle == "" {
		safeTitle = "anyflip-download"
	}

	pageCount, err := getPageCount(configjs)
	if err != nil {
		broadcastProgress(ProgressEvent{
			Stage:   "error",
			Error:   true,
			Message: "Gagal mendeteksi jumlah halaman: " + err.Error(),
		})
		return
	}

	pageFileNames := getPageFileNames(configjs)
	downloadURL, _ := url.Parse("https://online.anyflip.com/")

	fb := &flipbook{
		URL:       anyflipURL,
		title:     safeTitle,
		pageCount: pageCount,
	}

	if len(pageFileNames) == 0 {
		for i := 1; i <= pageCount; i++ {
			downloadURL.Path = path.Join(fb.URL.Path, "files", "mobile", strconv.Itoa(i)+".jpg")
			fb.pageURLs = append(fb.pageURLs, downloadURL.String())
		}
	} else {
		for i := 0; i < pageCount; i++ {
			downloadURL.Path = path.Join(fb.URL.Path, "files", "large", pageFileNames[i])
			fb.pageURLs = append(fb.pageURLs, downloadURL.String())
		}
	}

	broadcastProgress(ProgressEvent{
		Stage:   "downloading",
		Title:   bookTitle,
		Current: 0,
		Total:   pageCount,
		Message: fmt.Sprintf("Metadata ditemukan: %q (%d Halaman). Memulai download...", bookTitle, pageCount),
	})

	tempFolder, err := filepath.Abs(safeTitle)
	if err != nil {
		broadcastProgress(ProgressEvent{
			Stage:   "error",
			Error:   true,
			Message: "Folder unduhan tidak valid: " + err.Error(),
		})
		return
	}

	opts := downloadOptions{
		threads:    req.Threads,
		retries:    3,
		retryDelay: time.Second,
		onProgress: func(stage string, current int, total int, message string) {
			broadcastProgress(ProgressEvent{
				Stage:   stage,
				Title:   bookTitle,
				Current: current,
				Total:   total,
				Message: message,
			})
		},
	}

	err = fb.downloadImages(tempFolder, opts)
	if err != nil {
		broadcastProgress(ProgressEvent{
			Stage:   "error",
			Error:   true,
			Message: "Gagal mengunduh gambar: " + err.Error(),
		})
		return
	}

	broadcastProgress(ProgressEvent{
		Stage:   "converting",
		Title:   bookTitle,
		Current: 0,
		Total:   pageCount,
		Message: "Semua gambar selesai diunduh. Mengonversi ke PDF...",
	})

	cleanFileName := strings.ReplaceAll(bookTitle, "/", "-")
	cleanFileName = strings.ReplaceAll(cleanFileName, "\\", "-")
	cleanFileName = strings.ReplaceAll(cleanFileName, ":", "-")
	cleanFileName = strings.ReplaceAll(cleanFileName, "*", "")
	cleanFileName = strings.ReplaceAll(cleanFileName, "?", "")
	cleanFileName = strings.ReplaceAll(cleanFileName, "\"", "")
	cleanFileName = strings.ReplaceAll(cleanFileName, "<", "")
	cleanFileName = strings.ReplaceAll(cleanFileName, ">", "")
	cleanFileName = strings.ReplaceAll(cleanFileName, "|", "")
	cleanFileName = strings.TrimSpace(cleanFileName)
	if cleanFileName == "" {
		cleanFileName = safeTitle
	}
	outputFile := cleanFileName + ".pdf"
	_ = os.Remove(outputFile)

	err = createPDF(outputFile, tempFolder, req.ChunkSize, func(stage string, current int, total int, message string) {
		broadcastProgress(ProgressEvent{
			Stage:   stage,
			Title:   bookTitle,
			Current: current,
			Total:   total,
			Message: message,
		})
	})
	if err != nil {
		broadcastProgress(ProgressEvent{
			Stage:   "error",
			Error:   true,
			Message: "Gagal membuat file PDF: " + err.Error(),
		})
		return
	}

	if !req.KeepFolder {
		_ = os.RemoveAll(tempFolder)
	}

	broadcastProgress(ProgressEvent{
		Stage:   "completed",
		Title:   bookTitle,
		Current: pageCount,
		Total:   pageCount,
		Message: fmt.Sprintf("Sukses! File PDF tersimpan: %s", outputFile),
	})
}

func handleBooksAPI(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(".")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	books := []BookInfo{}
	for _, f := range files {
		if !f.IsDir() && strings.EqualFold(filepath.Ext(f.Name()), ".pdf") {
			info, err := f.Info()
			if err != nil {
				continue
			}

			sizeMB := float64(info.Size()) / (1024 * 1024)
			var sizeStr string
			if sizeMB >= 1.0 {
				sizeStr = fmt.Sprintf("%.1f MB", sizeMB)
			} else {
				sizeStr = fmt.Sprintf("%.1f KB", float64(info.Size())/1024)
			}

			books = append(books, BookInfo{
				Name:          f.Name(),
				SizeFormatted: sizeStr,
				Modified:      info.ModTime().Format("02/01/2006 15:04"),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func handleOpenFileAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cleanName := filepath.Base(payload.Filename)
	absPath, err := filepath.Abs(cleanName)
	if err != nil || !strings.HasSuffix(strings.ToLower(cleanName), ".pdf") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	exec.Command("cmd", "/c", "start", "", absPath).Start()
	w.WriteHeader(http.StatusOK)
}

func handleOpenFolderAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	absPath, _ := filepath.Abs(".")
	exec.Command("explorer.exe", absPath).Start()
	w.WriteHeader(http.StatusOK)
}
