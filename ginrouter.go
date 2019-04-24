package main

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
)

// Config holds all config vars
type Config struct {
	Addr        string `long:"http_addr" default:"localhost:8080"  description:"Http listen address"`
	UploadLimit int64  `long:"upload_limit" default:"8" description:"Upload size limit (Mb)"`
	ImgDir      string `long:"img_dir" default:"data/img" description:"Image upload destination"`
	ImgPath     string `long:"img_path" default:"/img" description:"Image URL path"`
	UploadPath  string `long:"upload_path" default:"/upload" description:"Image upload URL path"`
	PreviewDir  string `long:"preview_dir" default:"data/preview" description:"Image preview destination"`
	PreviewPath string `long:"preview_path" default:"/preview" description:"Image preview URL path"`
}

func setupConfig() (*Config, error) {
	cfg := &Config{}
	p := flags.NewParser(cfg, flags.Default)
	if _, err := p.Parse(); err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil, errors.New("ERR1") //os.Exit(1) // help printed
		} else {
			return nil, errors.New("ERR2") //os.Exit(2) // error message written already
		}
	}
	return cfg, nil
}

func setupRouter(cfg *Config) *gin.Engine {
	router := gin.Default()

	router.Static("/assets", "./assets")
	router.StaticFile("/favicon.ico", "./assets/favicon.ico")
	router.StaticFile("/", "./assets/index.html")
	router.StaticFS(cfg.ImgPath, http.Dir(cfg.ImgDir))

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	router.MaxMultipartMemory = cfg.UploadLimit << 20 // 8 MiB

	router.POST(cfg.UploadPath, func(c *gin.Context) {
		if c.ContentType() == "multipart/form-data" {
			handleMultiPart(cfg, c)
		} else {
			// POST with raw data: data:image/png;base64,...
			handleBase64(cfg, c)
		}
	})
	router.GET(cfg.UploadPath, func(c *gin.Context) {
		handleExternal(cfg, c)
	})
	router.GET(cfg.PreviewPath+"/:name", func(c *gin.Context) {
		handlePreview(cfg, c)
	})
	router.HEAD(cfg.PreviewPath+"/:name", func(c *gin.Context) {
		handlePreview(cfg, c)
	})

	return router
}

func handleMultiPart(cfg *Config, c *gin.Context) {
	form, _ := c.MultipartForm()
	name, err := uploadMultiPart(cfg.ImgDir, form)
	if err != nil {
		logError(c, err)
		return
	}
	c.Redirect(http.StatusFound, cfg.PreviewPath+"/"+*name)
}

func handleExternal(cfg *Config, c *gin.Context) {
	url := c.Query("url")
	name, err := uploadURL(cfg.ImgDir, url)
	if err != nil {
		logError(c, err)
		return
	}
	c.Redirect(http.StatusFound, cfg.PreviewPath+"/"+*name)
}

// handleBase64 stores raw base64 data and returns links in json
func handleBase64(cfg *Config, c *gin.Context) {
	buf, err := c.GetRawData()
	if err != nil {
		logError(c, err)
		return
	}
	name, err := uploadBase64(cfg.ImgDir, buf)
	if err != nil {
		logError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"file": cfg.ImgPath + "/" + *name, "preview": cfg.PreviewPath + "/" + *name})
}

func handlePreview(cfg *Config, c *gin.Context) {
	name := c.Param("name")
	file, err := getPreview(cfg.ImgDir, cfg.PreviewDir, name)
	if err != nil {
		if err.Error() == "404" {
			c.String(http.StatusNotFound, "Error: image '%s' not found", name)
		} else {
			logError(c, err)
		}
		return
	}
	c.File(*file)
}

func logError(c *gin.Context, e error) {
	c.String(http.StatusInternalServerError, "Error: %s", e.Error())
}
