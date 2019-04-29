// Package upload implements image upload handlers.
package upload

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"gopkg.in/birkirb/loggers.v1"
)

// Config holds all config vars
type Config struct {
	DownloadLimit int64  `long:"download_limit" default:"8" description:"External image size limit (Mb)"`
	Dir           string `long:"dir" default:"data/img" description:"Image upload destination"`
	PreviewDir    string `long:"preview_dir" default:"data/preview" description:"Preview image destination"`
	PreviewWidth  int    `long:"preview_width" default:"100" description:"Preview image width"`
	PreviewHeight int    `long:"preview_heigth" default:"100" description:"Preview image heigth"`
	UseRandomName bool   `long:"random_name" description:"Do not keep uploaded image filename"`
}

const (
	// ErrNoSingleFile returned when does not contain single file in field 'file'
	ErrNoSingleFile = "field 'file' does not contains single item"
	// ErrIncorrectData returned when field data does not contain valid base64 encoded data
	ErrIncorrectData = "incorrect data format"
	// ErrNotImage returned when media type isn't supported by underlying image processing package
	ErrNotImage = "Unsupported media type"
	// ErrNoCTypeExt returned when filename does not contain extension and we can't get it from content type
	ErrNoCTypeExt = "File ext for content type not found"
	// ErrFmtBadDownload returnd when download status != 200
	ErrFmtBadDownload = "Image download failed (%d)"

	// Base64MinCommaIndex holds minimal base64 image prefix len
	Base64MinCommaIndex = 21
)

// Service holds upload service
type Service struct {
	Config   *Config
	Log      loggers.Contextual
	getLimit int64 // store result of bytes to Mb calc
}

// New creates an Service object
func New(cfg Config, log loggers.Contextual) *Service {
	return &Service{&cfg, log, cfg.DownloadLimit << 20}
}

// HandleMultiPart stores image from multipart form
func (srv Service) HandleMultiPart(form *multipart.Form) (*string, error) {
	files, ok := form.File["file"]
	if !ok || len(files) != 1 {
		return nil, NewHTTPError(
			http.StatusBadRequest,
			errors.New(ErrNoSingleFile),
		)
	}
	file := files[0]
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	fileName := filepath.Base(file.Filename)
	name, err := srv.saveFile(src, contentType, fileName)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

// HandleURL reveives and stores image from URL
func (srv Service) HandleURL(url string) (*string, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, NewHTTPError(http.StatusServiceUnavailable, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, NewHTTPError(
			http.StatusServiceUnavailable,
			fmt.Errorf(ErrFmtBadDownload, response.StatusCode),
		)
	}

	src := io.LimitReader(response.Body, srv.getLimit)
	contentType := response.Header.Get("Content-Type")
	fileName := path.Base(response.Request.URL.Path)
	name, err := srv.saveFile(src, contentType, fileName)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

// HandleBase64 stores file received as base64 encoded string
func (srv Service) HandleBase64(data, name string) (*string, error) {
	prefixLen := strings.Index(data, ",")
	if prefixLen < Base64MinCommaIndex {
		return nil, NewHTTPError(
			http.StatusBadRequest,
			errors.New(ErrIncorrectData),
		)
	}
	contentType := strings.TrimSuffix(data[5:prefixLen], ";base64") // 5 means 'data:'
	file, err := base64.StdEncoding.DecodeString(data[prefixLen+1:])
	if err != nil {
		return nil, NewHTTPError(http.StatusBadRequest, err)
	}
	src := bytes.NewReader(file)
	name, err = srv.saveFile(src, contentType, name)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

// saveFile saves file from src and also creates preview for it
func (srv Service) saveFile(src io.Reader, contentType, fileName string) (name string, err error) {
	cfg := srv.Config

	dst, err := createFile(cfg.UseRandomName, cfg.Dir, contentType, fileName)
	defer func() {
		if err != nil {
			// remove image random dir if was created
			if dst != nil && path.Dir(dst.Name()) != cfg.Dir {
				e := os.Remove(path.Dir(dst.Name()))
				if e != nil {
					srv.Log.Errorf("Error removing file: ", e)
				}
			}
		}
	}()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			// remove image if exists
			e := os.Remove(dst.Name())
			if e != nil {
				srv.Log.Errorf("Error removing file: ", e)
			}
		}
	}()

	var cnt int64
	srcName := dst.Name()
	cnt, err = io.Copy(dst, src)
	dst.Close()
	if err != nil {
		return
	}

	// create preview
	var img image.Image
	img, err = imaging.Open(srcName)
	if err != nil {
		// File is not an image
		srv.Log.Warnf("Open error: %v", err)
		err = NewHTTPError(http.StatusUnsupportedMediaType, errors.New(ErrNotImage))
		return
	}
	name = strings.TrimPrefix(srcName, cfg.Dir)
	previewName := filepath.Join(cfg.PreviewDir, name)
	previewImage := imaging.Resize(img, cfg.PreviewWidth, cfg.PreviewHeight, imaging.Lanczos)

	// name may contains random dir, ensure dir exists anyway
	err = os.MkdirAll(path.Dir(previewName), 0700)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			// remove preview random dir if created
			if path.Dir(previewName) != cfg.PreviewDir {
				e := os.Remove(path.Dir(previewName))
				if e != nil {
					srv.Log.Errorf("Error removing preview dir: ", e)
				}
			}
		}
	}()
	err = imaging.Save(previewImage, previewName) // file mode allows read for all
	srv.Log.Infof("Saved %d of %s", cnt, srcName)
	return
}

// contentTypeExt returns first item from extension list for given content type
func contentTypeExt(contentType string) (ext string, err error) {
	var exts []string
	exts, err = mime.ExtensionsByType(contentType)
	if err != nil {
		return
	}
	if len(exts) == 0 {
		err = NewHTTPError(http.StatusUnsupportedMediaType, errors.New(ErrNoCTypeExt))
		return
	}
	ext = exts[0]
	return
}

// createFile creates and return handle of unique file
func createFile(useRandom bool, dir, contentType, fileName string) (dst *os.File, err error) {
	// Ensure dir exists
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return
	}
	if useRandom {
		// Generate random filename with original ext
		ext := path.Ext(fileName)
		if ext == "" {
			ext, err = contentTypeExt(contentType)
			if err != nil {
				return
			}
		}
		// create & lock file
		dst, err = ioutil.TempFile(dir, "*"+ext)
		return
	}
	// try to keep original filename
	ext := path.Ext(fileName)
	if ext == "" {
		// add ext from content type
		ext, err = contentTypeExt(contentType)
		if err != nil {
			return
		}
		fileName += ext
	}
	file := filepath.Join(dir, fileName)
	// Check if fileName is already used
	if _, err = os.Stat(file); err == nil {
		// file exists, add random dir
		var outDir string
		outDir, err = ioutil.TempDir(dir, "")
		if err != nil {
			return
		}
		file = filepath.Join(outDir, fileName)
	}
	// create & lock file
	dst, err = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	return
}
