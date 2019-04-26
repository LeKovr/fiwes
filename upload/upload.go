package upload

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
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
	KeepImageName bool   `long:"keep_name" description:"Keep uploaded image filename if given and correct"`
}

var ErrNotImage = errors.New("media not supported")

type Uploader struct {
	Config   *Config
	Log      loggers.Contextual
	getLimit int64
}

func New(cfg Config, log loggers.Contextual) *Uploader {
	return &Uploader{&cfg, log, cfg.DownloadLimit << 20}
}

func (u Uploader) MultiPart(form *multipart.Form) (*string, error) {
	files, ok := form.File["file"]
	if !ok || len(files) != 1 {
		return nil, errors.New("field 'file' is empty")
	}
	file := files[0]
	contentType := file.Header.Get("Content-Type")
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	name, err := u.saveFile(contentType, src)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

func (u Uploader) URL(url string) (*string, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Image download failed: " + response.Status)
	}
	src := io.LimitReader(response.Body, u.getLimit)
	contentType := response.Header.Get("Content-Type")
	name, err := u.saveFile(contentType, src)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

func (u Uploader) Base64(buf []byte) (*string, error) {
	data := string(buf)
	prefixLen := strings.Index(data, ",")
	if prefixLen < 5 {
		return nil, errors.New("incorrect data format")
	}
	contentType := strings.TrimSuffix(data[5:prefixLen], ";base64")
	file, err := base64.StdEncoding.DecodeString(data[prefixLen+1:])
	if err != nil {
		return nil, err
	}
	src := bytes.NewReader(file)
	name, err := u.saveFile(contentType, src)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

func (u Uploader) saveFile(contentType string, src io.Reader) (name string, err error) {
	cfg := u.Config
	var exts []string
	exts, err = mime.ExtensionsByType(contentType)
	if err != nil {
		return
	}
	if len(exts) == 0 {
		err = ErrNotImage
		return
	}
	var dst *os.File
	dst, err = ioutil.TempFile(cfg.Dir, "*"+exts[0])
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			e := os.Remove(dst.Name())
			if e != nil {
				u.Log.Errorf("Error removing file: ", e)
			}
		}
	}()

	var cnt int64
	cnt, err = io.Copy(dst, src)
	dst.Close()
	if err != nil {
		return
	}

	path := dst.Name()
	var img image.Image
	img, err = imaging.Open(path)
	if err != nil {
		u.Log.Warnf("Open error: %v", err)
		err = ErrNotImage
		return
	}

	imgPreview := imaging.Resize(img, cfg.PreviewWidth, cfg.PreviewHeight, imaging.Lanczos)

	name = filepath.Base(path)
	preview := filepath.Join(cfg.PreviewDir, name)
	err = imaging.Save(imgPreview, preview)
	u.Log.Infof("Saved %d of %s", cnt, path)
	return
}
