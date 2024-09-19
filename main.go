package main

import (
	"fmt"
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

type workspaceFolderHandler string

func (h workspaceFolderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean(r.URL.Path)

	var pathComponents []string

	dir, file := path.Split(cleanPath)
	if file != "" {
		pathComponents = append(pathComponents, file)
	}
	for dir != "/"+string(h)+"/" {
		dir, file = path.Split(dir[:len(dir)-1])
		pathComponents = append(pathComponents, file)
	}
	pathComponents = append(pathComponents, string(h), `C:\Users\Icosatess\Source`)
	slices.Reverse(pathComponents)
	fullpath := filepath.Join(pathComponents...)

	fi, fierr := os.Stat(fullpath)
	if fierr != nil {
		panic(fierr)
	}

	if fi.IsDir() {
		des, deserr := os.ReadDir(fullpath)
		if deserr != nil {
			panic(deserr)
		}
		w.Write([]byte(`
		<!DOCTYPE html>
		<title>Code server</title>
		<ul>
		`))
		for _, de := range des {
			currentPath := r.URL.Path
			deName := de.Name()
			deFullPath := path.Join(currentPath, deName)
			// TODO: XSS can happen here a lot
			w.Write([]byte(fmt.Sprintf(`<li><a href="%s">%s</a>`, deFullPath, deName)))
		}
		w.Write([]byte("</ul>"))
	} else {
		f, ferr := os.Open(fullpath)
		if ferr != nil {
			// TODO: handle file not found
			panic(ferr)
		}
		defer f.Close()

		w.Header().Set("Content-Type", "text/plain")
		io.Copy(w, f)
	}
}

func main() {
	http.HandleFunc("/", root)
	http.Handle("/minimapui/", workspaceFolderHandler("minimapui"))
	http.Handle("/minimapsrv/", workspaceFolderHandler("minimapsrv"))
	http.Handle("/minimapext/", workspaceFolderHandler("minimapext"))
	http.Handle("/codesrv/", workspaceFolderHandler("codesrv"))

	// TODO: serve as plain text
	// TODO: add a disallow-list for dotfiles and other stuff viewers shouldn't see
	log.Fatal(http.ListenAndServe("127.0.0.1:8081", nil))
}
