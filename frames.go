package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

type frameData struct {
	ContentPath string
}

var frameTemplate = template.Must(template.ParseFS(tplFS, "template/frame.html"))

func frameTest(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	rest, ok := strings.CutPrefix(p, "/frame")
	if !ok {
		panic("couldn't find /frame at start of path")
	}

	frameTemplate.Execute(w, frameData{
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
