package main

import (
	"net/http"
	"log"
	"sync"
	"html/template"
	"path/filepath"
	"flag"
	"os"
)

type templateHandler struct {
	once sync.Once
	filename string
	// templ represents a single template
	templ *template.Template
}

// ServeHTTP handles the HTTP request.
// ServeHTTPを持つことでHttp.Handlerに適合させる
// sync.Onceの値は常に同じものを使う必要があるので、pointer receiverである必要がある。
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	err := t.templ.Execute(w, r)
	if err != nil {
		log.Fatal("Template", err)
	}
}

func main() {
	var addr = flag.String("addr", ":8080", "Application Address")
	flag.Parse()

	r := newRoom(os.Stdout)
	// root pathへのリクエストを受ける
	http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/room",r)
	// start chat room
	go r.run()
	log.Println("Start web server. port:", *addr)
	// Start Server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe", err)
	}

}
