package main

import (
	"log"
	"net/http"
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

func main() {
	http.HandleFunc("/", root)
	http.Handle("/minimapui/", http.FileServer(http.Dir(`C:\Users\Icosatess\Source`)))
	http.Handle("/minimapsrv/", http.FileServer(http.Dir(`C:\Users\Icosatess\Source`)))
	http.Handle("/minimapext/", http.FileServer(http.Dir(`C:\Users\Icosatess\Source`)))
	http.Handle("/codesrv/", http.FileServer(http.Dir(`C:\Users\Icosatess\Source`)))

	// TODO: serve as plain text
	// TODO: add a disallow-list for dotfiles and other stuff viewers shouldn't see
	log.Fatal(http.ListenAndServe("127.0.0.1:8081", nil))
}
