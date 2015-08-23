package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"net/http"
	"html/template"
	"bytes"
	"regexp"
	"errors"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

type Page struct {
	Title string
	Body []byte
}

func markdowner(args ...interface{}) template.HTML {
	unsafe := blackfriday.MarkdownCommon([]byte(fmt.Sprintf("%s", args...)))
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return template.HTML(html)
}

var templateFuncs = template.FuncMap{"markdown": markdowner}

var templates = map[string]*template.Template {
	"edit": template.Must(template.New("edit").Funcs(templateFuncs).ParseFiles("tmpl/edit.html", "tmpl/layout.html")),
	"view": template.Must(template.New("view").Funcs(templateFuncs).ParseFiles("tmpl/view.html", "tmpl/layout.html")),
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("Invalid Page Title")
	}
	return m[2], nil
}

func (p *Page) save() error {
	filename := "data/" + p.Title + ".txt"
	body := bytes.TrimSpace(p.Body)
	return ioutil.WriteFile(filename, body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := "data/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}

	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}

	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}

	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates[tmpl].ExecuteTemplate(w, "layout", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}

		fn(w, r, m[2])
	}
}

func renderHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		viewHandler(w, r, "FrontPage")
	} else {
		http.NotFound(w, r)
		return
	}
}

func main() {
	fs := http.FileServer(http.Dir("static"))
  http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/", renderHome)

	err := http.ListenAndServe(":" + os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}
}
