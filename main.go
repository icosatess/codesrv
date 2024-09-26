package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
)

type directoryEntryInfo struct {
	FullPath string
	Name     string
}

const directoryListingTemplate = `
<!DOCTYPE html>
<title>Code server</title>
<ul>
{{range .}}
<li><a href="{{.FullPath}}">{{.Name}}</a>
{{end}}
</ul>`

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
	// This should only happen if the request is for the root path '/'
	if file != "" {
		pathComponents = append(pathComponents, file)
	}
	for dir != "/" {
		dir, file = path.Split(dir[:len(dir)-1])
		pathComponents = append(pathComponents, file)
	}
	pathComponents = append(pathComponents, `C:\Users\Icosatess\Source`)
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
		t, terr := template.New("directoryEntryTemplate").Parse(directoryListingTemplate)
		if terr != nil {
			panic(terr)
		}
		var deis []directoryEntryInfo
		for _, de := range des {
			currentPath := r.URL.Path
			deName := de.Name()
			deFullPath := path.Join(currentPath, deName)
			deis = append(deis, directoryEntryInfo{
				FullPath: deFullPath,
				Name:     deName,
			})
		}
		if err := t.Execute(w, deis); err != nil {
			panic(err)
		}
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
	srvAddr := "127.0.0.1:8080"
	log.Printf("Starting code server at %s", srvAddr)
	log.Fatal(http.ListenAndServe(srvAddr, nil))
}
