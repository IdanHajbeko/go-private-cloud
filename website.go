package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	MAX_UPLOAD_SIZE = 1024 * 1024 * 1024
	UPLOAD_DIR      = "./cloud/"
)

type PageData struct {
	Files []string
}

type Progress struct {
	TotalSize int64
	BytesRead int64
}

func (pr *Progress) Write(p []byte) (n int, err error) {
	n, err = len(p), nil
	pr.BytesRead += int64(n)
	pr.Print()
	return
}

func (pr *Progress) Print() {
	if pr.BytesRead == pr.TotalSize {
		return
	}
	fmt.Printf("File upload in progress: %d\n", pr.BytesRead)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Connection from IP: %s", r.RemoteAddr)
	files, err := ioutil.ReadDir(UPLOAD_DIR)
	if err != nil {
		log.Fatalf("Failed to read upload directory: %s", err)
	}

	var filenames []string
	for _, file := range files {
		filenames = append(filenames, file.Name())
	}

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Files: filenames,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["file"]

	for _, fileHeader := range files {
		log.Printf("Upload request from IP: %s, File: %s, Size: %d bytes", r.RemoteAddr, fileHeader.Filename, fileHeader.Size)
		if fileHeader.Size > MAX_UPLOAD_SIZE {
			http.Error(w, fmt.Sprintf("The uploaded file is too big: %s. Please use a file less than 1MB in size", fileHeader.Filename), http.StatusBadRequest)
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open uploaded file: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to open uploaded file %s: %s", fileHeader.Filename, err)
			return
		}
		defer file.Close()

		buff := make([]byte, 512)
		_, err = file.Read(buff)
		if err != nil && err != io.EOF {
			http.Error(w, "Failed to read uploaded file: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to read uploaded file %s: %s", fileHeader.Filename, err)
			return
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			http.Error(w, "Failed to seek to the beginning of the file: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to seek uploaded file %s to the beginning: %s", fileHeader.Filename, err)
			return
		}

		err = os.MkdirAll(UPLOAD_DIR, os.ModePerm)
		if err != nil {
			http.Error(w, "Failed to create upload directory: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to create upload directory %s: %s", UPLOAD_DIR, err)
			return
		}

		originalFilePath := filepath.Join(UPLOAD_DIR, fileHeader.Filename)
		filePath := originalFilePath

		for i := 1; ; i++ {
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				break
			}
			filePath = fmt.Sprintf("%s(%d)%s", strings.TrimSuffix(originalFilePath, filepath.Ext(originalFilePath)), i, filepath.Ext(originalFilePath))
		}

		f, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to create file: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to create file %s: %s", filePath, err)
			return
		}
		defer f.Close()
		log.Printf("File %s uploaded successfully", fileHeader.Filename)

		pr := &Progress{
			TotalSize: fileHeader.Size,
		}
		_, err = io.Copy(f, io.TeeReader(file, pr))
		if err != nil {
			http.Error(w, "Failed to write file: "+err.Error(), http.StatusInternalServerError)
			log.Printf("Failed to write file %s: %s", filePath, err)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Download request from IP: %s", r.RemoteAddr)
	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "File parameter is missing", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(UPLOAD_DIR, filename)
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open file: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to open file %s: %s", filePath, err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to get file info for %s: %s", filePath, err)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	log.Printf("Downloading file: %s", filename)

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Failed to send file: "+err.Error(), http.StatusInternalServerError)
		log.Printf("Failed to send file %s: %s", filePath, err)
		return
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", IndexHandler)
	mux.HandleFunc("/upload", uploadHandler)
	mux.HandleFunc("/download", downloadHandler)

	fmt.Println("Server listening on port 80...")
	if err := http.ListenAndServe(":80", mux); err != nil {
		log.Fatalf("Server error: %s", err)
	}
}
