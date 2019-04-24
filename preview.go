package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/disintegration/imaging"
)

var reFileName = regexp.MustCompile(`^\w+\.\w+$`)

func getPreview(root, previewDir, name string) (*string, error) {
	if !reFileName.Match([]byte(name)) {
		return nil, errors.New("Incorrect filename")
	}
	orig := filepath.Join(root, name)
	if _, err := os.Stat(orig); err == nil {
		// source image exists
		preview := filepath.Join(previewDir, name)
		if _, err := os.Stat(preview); err == nil {
			// return preview
			return &preview, nil
		} else if os.IsNotExist(err) {
			// TODO: send preview gen request
			// ...
			src, err := imaging.Open(orig)
			if err != nil {
				return nil, err
			}

			// Resize srcImage to size = 128x128px using the Lanczos filter.
			dst := imaging.Resize(src, 100, 100, imaging.Lanczos)
			err = imaging.Save(dst, preview)
			if err != nil {
				return nil, err
			}
			return &preview, nil
		} else {
			return nil, err
		}
	} else if os.IsNotExist(err) {
		// TODO		c.String(http.StatusNotFound, "Error: image '%s' not found", name)
		return nil, errors.New("404")
	} else {
		return nil, err
	}
}
