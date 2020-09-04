package static_file_server

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/moisespsena-go/task"
	defaultlogger "github.com/moisespsena-go/default-logger"
	yaml "gopkg.in/yaml.v2"

	"github.com/mitchellh/go-homedir"
	"github.com/moisespsena-go/httpu"
	"github.com/moisespsena-go/path-helpers"
)

type Server struct {
	*httpu.Server
	Config *Config
}

func NewServer(cfg *Config) *Server {
	srv := &Server{httpu.NewServer(&cfg.Config, cfg.CreateHandler()), cfg}
	srv.SetLog(defaultlogger.GetOrCreateLogger(path_helpers.GetCalledDir() + " SERVER"))
	srv.PreRun(func(ta task.Appender) error {
		var w bytes.Buffer
		yaml.NewEncoder(&w).Encode(srv.Config)
		srv.Log().Debug("Start With config:\n" + w.String())
		return nil
	})
	return srv
}

type Config struct {
	httpu.Config
	Prefix       string
	RootDir      string   `yaml:"root_dir"`
	CrossOrigins []string `yaml:"cross_origins"`
	AutoStart    bool     `yaml:"auto_start"`
}

func (cfg *Config) CreateServer() *Server {
	return NewServer(cfg)
}

func (cfg *Config) CreateHandler() http.Handler {
	dir, err := homedir.Expand(cfg.RootDir)
	if err != nil {
		panic(err)
	}
	var crossOrigin []string
	for _, co := range cfg.CrossOrigins {
		if co != "" {
			crossOrigin = append(crossOrigin, co)
		}
	}
	var co string
	if len(crossOrigin) > 0 {
		co = strings.Join(crossOrigin, " ")
	}
	return &Handler{cfg, http.FileServer(http.Dir(dir)), co}
}

type Handler struct {
	Config *Config
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
