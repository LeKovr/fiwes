package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func uploadMultiPart(root string, form *multipart.Form) (*string, error) {
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
	name, err := saveFile(root, contentType, src)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

func uploadURL(root, url string) (*string, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Image download failed: " + string(response.StatusCode))
	}
	src := response.Body
	contentType := response.Header.Get("Content-Type")
	name, err := saveFile(root, contentType, src)
	if err != nil {
		return nil, err
	}
	return &name, nil

}

func uploadBase64(root string, buf []byte) (*string, error) {
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
	name, err := saveFile(root, contentType, src)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

func saveFile(root, contentType string, src io.Reader) (name string, err error) {
	var exts []string
	exts, err = mime.ExtensionsByType(contentType)
	if err != nil {
		return
	}
	if len(exts) == 0 {
		err = errors.New("Unsupported content type")
		return
	}
	var dst *os.File
	dst, err = ioutil.TempFile(root, "*"+exts[0])
	if err != nil {
		return
	}
	defer dst.Close()
	var cnt int64
	cnt, err = io.Copy(dst, src)
	if err != nil {
		return
	}
	path := dst.Name()
	log.Printf("Saved %d of %s", cnt, path)
	name = filepath.Base(path)
	return
}
