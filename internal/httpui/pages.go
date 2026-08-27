package httpui

import "net/http"

func (s *Server) Root(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/workspace", http.StatusTemporaryRedirect)
}
func (s *Server) Workspace(w http.ResponseWriter, r *http.Request) {
	b, e := webFS.ReadFile("web/index.html")
	if e != nil {
		http.Error(w, "页面不可用", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}
func (s *Server) AssetCSS(w http.ResponseWriter, r *http.Request) {
	b, e := webFS.ReadFile("web/app.css")
	if e != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(b)
}
func (s *Server) AssetJS(w http.ResponseWriter, r *http.Request) {
	b, e := webFS.ReadFile("web/app.js")
	if e != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(b)
}
