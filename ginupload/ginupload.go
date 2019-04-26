package ginupload

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"gopkg.in/birkirb/loggers.v1"

	"imgserv/upload"
)

// Config holds all config vars
type Config struct {
	upload.Config
	Path        string `long:"path" default:"/img" description:"Image URL path"`
	UploadPath  string `long:"upload_path" default:"/upload" description:"Image upload URL path"`
	PreviewPath string `long:"preview_path" default:"/preview" description:"Preview image URL path"`
}

type GinUpload struct {
	Config Config
	up     *upload.Uploader
}

func New(cfg Config, log loggers.Contextual) *GinUpload {
	return &GinUpload{cfg, upload.New(cfg.Config, log)}
}

func (gu GinUpload) MultiPart(c *gin.Context) {
	form, _ := c.MultipartForm()
	name, err := gu.up.MultiPart(form)
	if err != nil {
		logError(c, err)
		return
	}
	c.Redirect(http.StatusFound, gu.Config.PreviewPath+"/"+*name)
}

func (gu GinUpload) External(c *gin.Context) {
	url := c.Query("url")
	name, err := gu.up.URL(url)
	if err != nil {
		logError(c, err)
		return
	}
	c.Redirect(http.StatusFound, gu.Config.PreviewPath+"/"+*name)
}

// Base64 reads POST with raw data (data:image/png;base64,...) and returns links in json
func (gu GinUpload) Base64(c *gin.Context) {
	buf, err := c.GetRawData()
	if err != nil {
		logError(c, err)
		return
	}
	name, err := gu.up.Base64(buf)
	if err != nil {
		logError(c, err)
		return
	}
	cfg := gu.Config
	c.JSON(http.StatusOK, gin.H{"file": cfg.Path + "/" + *name, "preview": cfg.PreviewPath + "/" + *name})
}

func logError(c *gin.Context, e error) {
	log.Printf("ERR: %s", e)
	if e == upload.ErrNotImage {
		c.String(http.StatusUnsupportedMediaType, "Unsupported media type")
	} else {
		c.String(http.StatusInternalServerError, "Error: %s", e.Error())
	}
}
