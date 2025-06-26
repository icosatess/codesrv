package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"io/fs"
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

//go:embed template
var tplFS embed.FS

// const sourceRoot = `C:\Users\Icosatess\Source`

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
	if errors.Is(rootFileErr, fs.ErrNotExist) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if rootFileErr != nil {
		log.Printf("got non-not-found error reading root template path from FS, returning 404 Not Found: %v", rootFileErr)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if n, err := w.Write(rootFile); err != nil {
		log.Printf("failed to fully write root file after %d bytes: %v", n, err)
		return
	}
}

func serveWorkspaceFolder(rootPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cleanPath := path.Clean(r.URL.Path)

		rem := cleanPath
		for {
			d, f := path.Split(rem)
			if f != "" {
				if slices.Contains(disallowedFilenames, f) {
					http.Error(w, "Icosatess has disallowed public viewing of this file/folder", http.StatusForbidden)
					return
				}
				rem = d
				continue
			}

			if d == "" || d == "/" {
				// Empty or root path. Allow.
				break
			} else {
				// Remove trailing slash and try again.
				rem = d[:len(d)-1]
				continue
			}
		}

		relativePath := filepath.FromSlash(cleanPath)
		fullPath := filepath.Join(rootPath, relativePath)

		fi, fierr := os.Stat(fullPath)
		if errors.Is(fierr, fs.ErrNotExist) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if fierr != nil {
			log.Printf("got non-not-found error stat-ing destination path %s, returning 404 Not Found: %v", fullPath, fierr)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		if fi.IsDir() {
			serveWorkspaceFolderDirectory(w, r, cleanPath, fullPath)
		} else {
			serveWorkspaceFolderSourceFile(w, cleanPath, fullPath)
		}
	}
}

func serveWorkspaceFolderDirectory(w http.ResponseWriter, r *http.Request, cleanPath, fullPath string) {
	des, deserr := os.ReadDir(fullPath)
	if deserr != nil {
		if len(des) == 0 {
			log.Printf("got error reading directory %s with no entries, returning 404 Not Found: %v", fullPath, deserr)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			log.Printf("got error reading directory %s with partial entries, continuing: %v", fullPath, deserr)
		}
	}

	var deis []directoryEntryInfo
	for _, de := range des {
		currentPath := r.URL.Path
		deName := de.Name()
		if slices.Contains(disallowedFilenames, deName) {
			continue
		}
		deFullPath := path.Join(currentPath, deName)
		deis = append(deis, directoryEntryInfo{
			FullPath: deFullPath,
			Name:     deName,
		})
	}

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	if err := directoryListingTemplate.Execute(w, directoryListingData{
		Path:      cleanPath,
		ParentDir: path.Dir(cleanPath),
		Entries:   deis,
	}); err != nil {
		log.Printf("failed to execute directory file template, giving up: %v", err)
		return
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

type Config struct {
	WorkspaceFolders map[string]string `json:"workspaceFolders"`
}

func main() {
	f, ferr := os.Open("config.json")
	if ferr != nil {
		log.Fatal("couldn't open configuration from config.json")
	}

	bs, bserr := io.ReadAll(f)
	if bserr != nil {
		log.Fatal("couldn't read configuration from config.json")
	}

	var cfg Config
	if err := json.Unmarshal(bs, &cfg); err != nil {
		log.Fatal("couldn't parse configuration from config.json")
	}

	http.HandleFunc("/", root)
	for k, v := range cfg.WorkspaceFolders {
		http.HandleFunc(path.Join("/", k)+"/", serveWorkspaceFolder(v))
	}

	srvAddr := os.Getenv("CODE_SERVER_ADDRESS")
	log.Printf("Starting code server at %s", srvAddr)
	log.Fatal(http.ListenAndServe(srvAddr, nil))
}
