package main

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

//go:embed template
var tplFS embed.FS

const sourceRoot = `C:\Users\Icosatess\Source`

type directoryEntryInfo struct {
	FullPath string
	Name     string
}

type directoryListingData struct {
	Path      string
	ParentDir string
	Entries   []directoryEntryInfo
}

var directoryListingTemplate = template.Must(template.ParseFS(tplFS, "template/directory.html"))

type sourceFileData struct {
	FilePath string
	Body     template.HTML
}

var sourceFileTemplate = template.Must(template.ParseFS(tplFS, "template/source.html"))

func root(w http.ResponseWriter, r *http.Request) {
	rootFile, rootFileErr := tplFS.ReadFile("template/root.html")
	if rootFileErr != nil {
		// Treat all errors as if file does not exist.
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if n, err := w.Write(rootFile); err != nil {
		log.Printf("failed to fully write root file after %d bytes: %v", n, err)
		return
	}
}

func serveWorkspaceFolder(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean(r.URL.Path)

	filename := path.Base(cleanPath)
	if filename == "secrets.json" {
		http.Error(w, "Icosatess has disallowed public viewing of this file", http.StatusForbidden)
		return
	}

	relativePath := filepath.FromSlash(cleanPath)
	fullPath := filepath.Join(sourceRoot, relativePath)

	fi, fierr := os.Stat(fullPath)
	if fierr != nil {
		panic(fierr)
	}

	if fi.IsDir() {
		serveWorkspaceFolderDirectory(w, r, cleanPath, fullPath)
	} else {
		serveWorkspaceFolderSourceFile(w, cleanPath, fullPath)
	}
}

func serveWorkspaceFolderDirectory(w http.ResponseWriter, r *http.Request, cleanPath, fullPath string) {
	des, deserr := os.ReadDir(fullPath)
	if deserr != nil {
		panic(deserr)
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
	if err := directoryListingTemplate.Execute(w, directoryListingData{
		Path:      cleanPath,
		ParentDir: path.Dir(cleanPath),
		Entries:   deis,
	}); err != nil {
		panic(err)
	}
}

func serveWorkspaceFolderSourceFile(w http.ResponseWriter, cleanPath, fullPath string) {
	f, ferr := os.Open(fullPath)
	if errors.Is(ferr, fs.ErrNotExist) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if ferr != nil {
		log.Printf("got non-not-found error opening source file, returning 404 Not Found: %v", ferr)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	defer f.Close()
	text, textErr := io.ReadAll(f)
	if textErr != nil {
		log.Printf("got error reading source file, returning 404 Not Found: %v", ferr)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	lexer := lexers.Match(fullPath)
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
		log.Printf("failed to tokenize source file, returning plain text: %v", iteratorErr)
		w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		w.Write(text)
		return
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		log.Printf("failed to format source file, returning plain text: %v", err)
		w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		w.Write(text)
		return
	}

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	if err := sourceFileTemplate.Execute(w, sourceFileData{
		FilePath: cleanPath,
		Body:     template.HTML(buf.Bytes()),
	}); err != nil {
		log.Printf("failed to execute source file template, giving up: %v", err)
		return
	}
}

func main() {
	http.HandleFunc("/", root)
	http.HandleFunc("/minimapui/", serveWorkspaceFolder)
	http.HandleFunc("/minimapsrv/", serveWorkspaceFolder)
	http.HandleFunc("/minimapext/", serveWorkspaceFolder)
	http.HandleFunc("/codesrv/", serveWorkspaceFolder)
	http.HandleFunc("/chatbot/", serveWorkspaceFolder)
	http.HandleFunc("/sidebar/", sidebar)
	http.HandleFunc("/frame/", frameTest)

	// TODO: serve as plain text
	// TODO: add a disallow-list for dotfiles and other stuff viewers shouldn't see
	srvAddr := "127.0.0.1:8080"
	log.Printf("Starting code server at %s", srvAddr)
	log.Fatal(http.ListenAndServe(srvAddr, nil))
}
