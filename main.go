package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

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
:root {
	color-scheme: light dark;
}

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

	// TODO: replace all of this with filepath.FromSlash
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

const frameTemplate = `
<!DOCTYPE html>
<meta charset="UTF-8">
<title>Icosatess</title>
<style>
:root {
	color-scheme: light dark;
}
</style>

<frameset cols="20%, 80%">
<frame src="/sidebar">
<frame src="{{.ContentPath}}" name="contentpane">
</frameset>
<noframes>
no frames content here
</noframes>
`

func frameTest(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	rest, ok := strings.CutPrefix(p, "/frame")
	if !ok {
		panic("couldn't find /frame at start of path")
	}

	t, terr := template.New("frameTest").Parse(frameTemplate)
	if terr != nil {
		panic(terr)
	}
	t.Execute(w, struct {
		ContentPath string
	}{
		ContentPath: rest,
	})
}

type parentDirectory struct {
	RelativePath string
	Path         string
	Items        []*parentDirectory
}

func sidebar(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html>
<meta charset="UTF-8">
<title>Sidebar</title>
<style>
:root {
	color-scheme: light dark;
}
</style>
<ul>
`)
	basePath := `C:\Users\Icosatess\Source\codesrv`
	var parents []*parentDirectory = []*parentDirectory{{
		Path: basePath,
	}}
	filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		log.Printf("walkdir callback called with path=%s, d=%+v, err=%+v", path, d, err)
		log.Printf("current state of parents is size %d: %+v", len(parents), parents)
		if err != nil {
			log.Printf("error walking dir %s in %s: %v", d, path, err)
			panic(err)
		}

		dir := filepath.Dir(path)
		for i := len(parents) - 1; i >= 0; i-- {
			if parents[i].Path == dir {
				parents = parents[:i+1]
				after, ok := strings.CutPrefix(path, basePath)
				if !ok {
					panic("couldn't cut the base path out")
				}
				pd := parentDirectory{Path: path, RelativePath: after}
				parents[i].Items = append(parents[i].Items, &pd)
				if d.IsDir() {
					parents = append(parents, &pd)
				}
				break
			}
		}
		return nil
	})
	// buf.WriteString(fmt.Sprintf("<h1>%s</h1>", path))
	DoRecursionInto(&buf, "/codesrv", *parents[0])
	buf.WriteString("</ul>")
	w.Write(buf.Bytes())
}

func DoRecursionInto(buf *bytes.Buffer, prefix string, pd parentDirectory) {
	log.Printf("DoRecursionInto called for %s", pd.Path)
	log.Printf("It has items %+v", pd.Items)
	urlPath := path.Join(prefix, filepath.ToSlash(pd.RelativePath))
	buf.WriteString(fmt.Sprintf(`<li><a href="%s" target="contentpane">%s</a>`, urlPath, filepath.Base(pd.Path)))
	if len(pd.Items) == 0 {
		return
	}
	buf.WriteString("<ul>")
	for _, item := range pd.Items {
		DoRecursionInto(buf, prefix, *item)
	}
	buf.WriteString("</ul>")
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
