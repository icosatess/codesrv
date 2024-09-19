package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
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

	var pathComponents []string

	dir, file := path.Split(cleanPath)
	pathComponents = append(pathComponents, file)
	for dir != "/codesrv/" {
		dir, file = path.Split(dir[:len(dir)-1])
		pathComponents = append(pathComponents, file)
	}
	pathComponents = append(pathComponents, `C:\Users\Icosatess\Source\codesrv`)
	slices.Reverse(pathComponents)
	fullpath := filepath.Join(pathComponents...)
	log.Printf("the path is: %v", pathComponents)

	f, ferr := os.Open(fullpath)
	if ferr != nil {
		// TODO: handle file not found
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
