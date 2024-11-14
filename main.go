package main

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

type directoryEntryInfo struct {
	FullPath string
	Name     string
}

type directoryListingData struct {
	Path      string
	ParentDir string
	Entries   []directoryEntryInfo
}

const directoryListingTemplate = `
<!DOCTYPE html>
<meta charset="UTF-8">
<title>Icosatess</title>
<style>
:root {
	color-scheme: light dark;
}
</style>
<h1>{{.Path}}</h1>
<ul>
<li><a href="{{.ParentDir}}">..</a>
{{range .Entries}}
<li><a href="{{.FullPath}}">{{.Name}}</a>
{{end}}
</ul>`

type sourceFileData struct {
	FilePath string
	Body     template.HTML
}

const sourceFileTemplate = `
<!DOCTYPE html>
<meta charset="UTF-8">
<title>{{.FilePath}}</title>
<style>
body, pre {
	margin: 0;
}
</style>
{{.Body}}
</ul>`

const rootIndex = `
<!DOCTYPE html>
<meta charset="UTF-8">
<title>Icosatess</title>
<style>
:root {
	color-scheme: light dark;
}
</style>
<h1>Icosatess’s code</h1>
<ul>
<li><a href="/minimapui">minimapui</a>
<li><a href="/minimapsrv">minimapsrv</a>
<li><a href="/minimapext">minimapext</a>
<li><a href="/codesrv">codesrv</a>
<li><a href="/chatbot">chatbot</a>
</ul>
`

func root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(rootIndex))
}

func serveWorkspaceFolder(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean(r.URL.Path)

	var pathComponents []string

	dir, file := path.Split(cleanPath)
	if file == "secrets.json" {
		http.Error(w, "Icosatess has disallowed public viewing of this file", http.StatusForbidden)
		return
	}
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
		if err := t.Execute(w, directoryListingData{
			Path:      cleanPath,
			ParentDir: path.Dir(cleanPath),
			Entries:   deis,
		}); err != nil {
			panic(err)
		}
	} else {
		f, ferr := os.Open(fullpath)
		if ferr != nil {
			// TODO: handle file not found
			panic(ferr)
		}
		text, textErr := io.ReadAll(f)
		if textErr != nil {
			panic(textErr)
		}
		if err := f.Close(); err != nil {
			panic(err)
		}

		lexer := lexers.Match(fullpath)
		if lexer == nil {
			lexer = lexers.Fallback
		}

		style := styles.Get("github-dark")
		if style == nil {
			style = styles.Fallback
		}
		formatter := html.New(html.WithLineNumbers(true))

		iterator, iteratorErr := lexer.Tokenise(nil, string(text))
		if iteratorErr != nil {
			panic(iteratorErr)
		}

		w.Header().Set("Content-Type", "text/html;charset=UTF-8")
		var buf bytes.Buffer
		if err := formatter.Format(&buf, style, iterator); err != nil {
			panic(err)
		}
		t, terr := template.New("sourceFileTemplate").Parse(sourceFileTemplate)
		if terr != nil {
			panic(terr)
		}
		t.Execute(w, sourceFileData{
			FilePath: cleanPath,
			Body:     template.HTML(buf.Bytes()),
		})
	}
}

func main() {
	http.HandleFunc("/", root)
	http.HandleFunc("/minimapui/", serveWorkspaceFolder)
	http.HandleFunc("/minimapsrv/", serveWorkspaceFolder)
	http.HandleFunc("/minimapext/", serveWorkspaceFolder)
	http.HandleFunc("/codesrv/", serveWorkspaceFolder)
	http.HandleFunc("/chatbot/", serveWorkspaceFolder)

	// TODO: serve as plain text
	// TODO: add a disallow-list for dotfiles and other stuff viewers shouldn't see
	srvAddr := "127.0.0.1:8080"
	log.Printf("Starting code server at %s", srvAddr)
	log.Fatal(http.ListenAndServe(srvAddr, nil))
}
