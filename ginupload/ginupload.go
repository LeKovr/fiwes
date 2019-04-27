// Package ginupload implements gin handlers for file upload processing
package ginupload

//go:generate moq -out upload_moq_test.go . Uploader

import (
	"log"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/birkirb/loggers.v1"

	"fiwes/upload"
)

// Config holds all config vars
type Config struct {
	upload.Config
	Path        string `long:"path" default:"/img" description:"Image URL path"`
	UploadPath  string `long:"upload_path" default:"/upload" description:"Image upload URL path"`
	PreviewPath string `long:"preview_path" default:"/preview" description:"Preview image URL path"`
}

// Uploader holds methods of underlying upload package
type Uploader interface {
	HandleMultiPart(form *multipart.Form) (*string, error)
	HandleURL(url string) (*string, error)
	HandleBase64(data, name string) (*string, error)
}

// Service holds ginupload service
type Service struct {
	Config Config
	up     Uploader
}

// New creates an Service object
func New(cfg Config, log loggers.Contextual, upl Uploader) *Service {
	if upl == nil {
		upl = upload.New(cfg.Config, log)
	}
	return &Service{cfg, upl}
}

// HandleMultiPart handles a file received as multipart form
func (srv Service) HandleMultiPart(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		logError(c, err)
		return
	}
	name, err := srv.up.HandleMultiPart(form)
	if err != nil {
		logError(c, err)
		return
	}
	c.Redirect(http.StatusFound, srv.Config.PreviewPath+*name)
}

// HandleURL handles an image from url field
func (srv Service) HandleURL(c *gin.Context) {
	url := c.Query("url")
	name, err := srv.up.HandleURL(url)
	if err != nil {
		logError(c, err)
		return
	}
	c.Redirect(http.StatusFound, srv.Config.PreviewPath+*name)
}

// File hold JSON request struct
type File struct {
	Name string `form:"name" json:"name" binding:"required"`
	Data string `form:"data" json:"data" binding:"required"`
}

// HandleBase64 reads POST with JSON data (data:image/png;base64,...) and returns JSON with links to file and preview
func (srv Service) HandleBase64(c *gin.Context) {
	var json File
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name, err := srv.up.HandleBase64(json.Data, json.Name)
	if err != nil {
		logError(c, err)
		return
	}
	cfg := srv.Config
	c.JSON(http.StatusOK, gin.H{"file": cfg.Path + *name, "preview": cfg.PreviewPath + *name})
}

// logError fills response with error message
func logError(c *gin.Context, e error) {
	log.Printf("ERR: %s", e)
	if e == upload.ErrNotImage {
		c.String(http.StatusUnsupportedMediaType, "Unsupported media type")
	} else {
		c.String(http.StatusInternalServerError, "Error: %s", e.Error())
	}
}
