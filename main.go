package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`
	<!DOCTYPE html>
	<title>Code server</title>
	<ul>
	<li><a href="/minimapui">minimapui</a>
	<li><a href="/minimapsrv">minimapsrv</a>
	<li><a href="/minimapext">minimapext</a>
	<li><a href="/codesrv">codesrv</a>
	</ul>
	`))
}

func codesrv(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean(r.URL.Path)
	dir, file := path.Split(cleanPath)
	_ = dir
	fullpath := filepath.Join(`C:\Users\Icosatess\Source\codesrv`, file)
	f, ferr := os.Open(fullpath)
	if ferr != nil {
		panic(ferr)
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/plain")
	io.Copy(w, f)
}

func main() {
	http.HandleFunc("/", root)
	http.Handle("/minimapui/", http.FileServer(http.Dir(`C:\Users\Icosatess\Source`)))
	http.Handle("/minimapsrv/", http.FileServer(http.Dir(`C:\Users\Icosatess\Source`)))
	http.Handle("/minimapext/", http.FileServer(http.Dir(`C:\Users\Icosatess\Source`)))
	http.HandleFunc("/codesrv/", codesrv)

	// TODO: serve as plain text
	// TODO: add a disallow-list for dotfiles and other stuff viewers shouldn't see
	log.Fatal(http.ListenAndServe("127.0.0.1:8081", nil))
}
