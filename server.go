package static_file_server

import (
	"net/http"
	"strings"

	"github.com/mitchellh/go-homedir"
)

type Config struct {
	Addr         string
	Prefix       string
	RootDir      string
	CrossOrigins []string
	AutoStart    bool
}

type Handler struct {
	*Server
	fileHandler http.Handler
	CrossOrigin string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.CrossOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", h.CrossOrigin)
	}
	if h.Config.Prefix != "" {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, h.Config.Prefix)
	}

	h.fileHandler.ServeHTTP(w, r)
}

type Server struct {
	Config Config
}

func NewServer(config Config) *Server {
	return &Server{config}
}

func (s *Server) Handler() http.Handler {
	dir, err := homedir.Expand(s.Config.RootDir)
	if err != nil {
		panic(err)
	}
	var crossOrigin []string
	for _, co := range s.Config.CrossOrigins {
		if co != "" {
			crossOrigin = append(crossOrigin, co)
		}
	}
	var co string
	if len(crossOrigin) > 0 {
		co = strings.Join(crossOrigin, " ")
	}
	return &Handler{s, http.FileServer(http.Dir(dir)), co}
}

func (s *Server) LisenAndServer() {
	http.ListenAndServe(s.Config.Addr, s.Handler())
}
