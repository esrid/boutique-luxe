// Package render loads html/template pages against a shared base layout.
// Every page is parsed once (layout + page clone, so {{define}} blocks in
// one page can't collide with another) and cached; ExecuteTemplate("layout")
// then streams the response.
package render

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sync"
	"time"
)

// Layout carries the fields every storefront page template needs regardless
// of its own content (footer year, mini-cart badge). Page-specific data
// structs embed this.
type Layout struct {
	Year      int
	CartCount int
}

func NewLayout(cartCount int) Layout {
	return Layout{Year: time.Now().Year(), CartCount: cartCount}
}

type Renderer struct {
	fs         fs.FS
	layoutGlob string
	funcs      template.FuncMap

	mu    sync.RWMutex
	cache map[string]*template.Template
}

func New(templatesFS fs.FS, layoutGlob string, funcs template.FuncMap) *Renderer {
	return &Renderer{
		fs:         templatesFS,
		layoutGlob: layoutGlob,
		funcs:      funcs,
		cache:      make(map[string]*template.Template),
	}
}

func (r *Renderer) page(name string) (*template.Template, error) {
	r.mu.RLock()
	tmpl, ok := r.cache[name]
	r.mu.RUnlock()
	if ok {
		return tmpl, nil
	}

	tmpl, err := template.New("layout").Funcs(r.funcs).ParseFS(r.fs, r.layoutGlob)
	if err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}
	tmpl, err = tmpl.ParseFS(r.fs, name)
	if err != nil {
		return nil, fmt.Errorf("parse page %s: %w", name, err)
	}

	r.mu.Lock()
	r.cache[name] = tmpl
	r.mu.Unlock()
	return tmpl, nil
}

// Render executes the named page template (its "content" define, plus any
// "title"/"head_extra"/"scripts" overrides) inside the base "layout".
//
// ponytail: parse errors surface before any bytes are written, so callers
// can still send a proper error response. An ExecuteTemplate failure
// mid-stream (after headers are sent) can't be un-sent — only logged. Add
// a buffering render step if that ever needs to become a clean 500 too.
func (r *Renderer) Render(w http.ResponseWriter, status int, page string, data any) error {
	tmpl, err := r.page(page)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	return tmpl.ExecuteTemplate(w, "layout", data)
}
