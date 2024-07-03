package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1>Download</h1>")
		files, err := ioutil.ReadDir("./cloud/")
		if err != nil {
			log.Fatal(err)
		}
		count := 0
		for _, f := range files {
			count++
			fmt.Fprintf(w, strconv.Itoa(count)+". ")
			fmt.Fprintf(w, `<button onclick="document.location.href = '`+"download/?file="+f.Name()+`'";>`+f.Name()+"</button><br>")
		}
		fmt.Fprintf(w, "<button> upload </button>")
	})

	http.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		files, err := ioutil.ReadDir("./cloud/")
		if err != nil {
			log.Fatal(err)
		}
		var success bool
		file := r.URL.Query().Get("file")
		for _, f := range files {
			if f.Name() == file {
				w.Header().Set("Content-Disposition", "attachment; filename="+f.Name())
				w.Header().Set("Content-Type", "application/octet-stream")
				http.ServeFile(w, r, "./cloud/"+f.Name())
				success = true
			}
		}
		if file == "" {
			fmt.Fprintf(w, "<h1>This page is for downloading files add to the url ?file={file name}"+file+"</h1>")
		} else if !success {
			fmt.Fprintf(w, "<h1>There is no file named "+file+"</h1>")
		}
	})

	if err := http.ListenAndServe(":80", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
