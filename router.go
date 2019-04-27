package main

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"

	mapper "github.com/birkirb/loggers-mapper-logrus"
	"github.com/sirupsen/logrus"
	"gopkg.in/birkirb/loggers.v1"

	"fiwes/ginupload"
)

// Config holds all config vars
type Config struct {
	Addr        string `long:"http_addr" default:"localhost:8080"  description:"Http listen address"`
	UploadLimit int64  `long:"upload_limit" default:"8" description:"Upload size limit (Mb)"`
	ShowHTML    bool   `long:"html" description:"Show html index page"`

	Img ginupload.Config `group:"Image upload Options" namespace:"img"`
}

// ErrGotHelp returned after showing requested help
var ErrGotHelp = errors.New("help printed")

// setupConfig loads flags from args (if given) or command flags and ENV otherwise
func setupConfig(args ...string) (*Config, error) {
	cfg := &Config{}
	p := flags.NewParser(cfg, flags.Default)
	var err error
	if len(args) == 0 {
		_, err = p.Parse()
	} else {
		_, err = p.ParseArgs(args)
	}
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil, ErrGotHelp
		}
		return nil, err // error message printed already
	}
	return cfg, nil
}

// setupLog creates logger
func setupLog() loggers.Contextual {
	l := logrus.New()
	if gin.IsDebugging() {
		l.SetLevel(logrus.DebugLevel)
		l.SetReportCaller(true)
	}
	return &mapper.Logger{Logger: l} // Same as mapper.NewLogger(l) but without info log message
}

// setupRouter creates gin router
func setupRouter(cfg *Config, log loggers.Contextual) *gin.Engine {
	router := gin.Default()
	if cfg.ShowHTML {
		router.Static("/static", "./assets/static")
		router.StaticFile("/favicon.ico", "./assets/favicon.ico")
		router.StaticFile("/", "./assets/index.html")
	}
	router.StaticFS(cfg.Img.Path, http.Dir(cfg.Img.Dir))
	router.StaticFS(cfg.Img.PreviewPath, http.Dir(cfg.Img.PreviewDir))

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	router.MaxMultipartMemory = cfg.UploadLimit << 20 // 8 MiB

	gup := ginupload.New(cfg.Img, log, nil)

	router.POST(cfg.Img.UploadPath, func(c *gin.Context) {
		if c.ContentType() == "multipart/form-data" {
			gup.HandleMultiPart(c)
		} else {
			gup.HandleBase64(c)
		}
	})
	router.GET(cfg.Img.UploadPath, func(c *gin.Context) {
		gup.HandleURL(c)
	})
	return router
}
