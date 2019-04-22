package main

import (
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
)

// Config holds all config vars
type Config struct {
	Addr       string `long:"http_addr" default:"localhost:8080"  description:"Http listen address"`
	BufferSize int    `long:"buffer_size" default:"64" description:"Template buffer size"`
	ImgDir     string `long:"img_dir" default:"data/img" description:"Image upload destination"`
	PreviewDir string `long:"preview_dir" default:"data/preview" description:"Image preview destination"`
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
	router.StaticFile("/favicon.ico", "./resources/favicon.ico")
	router.StaticFS("/img", http.Dir("./data/img"))

	var name string
	// Set a lower memory limit for multipart forms (default is 32 MiB)
	// router.MaxMultipartMemory = 8 << 20  // 8 MiB
	router.POST("/upload", func(c *gin.Context) {

		log.Println("***inup***")

		if c.ContentType() == "multipart/form-data" {
			form, _ := c.MultipartForm()
			files, ok := form.File["file"]
			if !ok || len(files) != 1 {
				// return error
				log.Println("no data")
			}
			file := files[0]
			ext := filepath.Ext(file.Filename) // TODO: other way to get content type?

			dst, err := ioutil.TempFile(cfg.ImgDir, "*"+ext)
			if err != nil {
				logError(c, err)
				return
			}
			defer dst.Close()
			// Upload the file to specific dst.
			src, err := file.Open()
			if err != nil {
				logError(c, err)
				return
			}
			defer src.Close()
			cnt, err := io.Copy(dst, src)
			if err != nil {
				logError(c, err)
				return
			}
			name = dst.Name()
			log.Printf("Saved %d of %s", cnt, name)
		} else {
			// data:image/png;base64,...
			// todo: check format
			buf, err := c.GetRawData()
			if err != nil {
				logError(c, err)
				return
			}

			/*			buf := make([]byte, c.Request.)
						num, _ := c.Request.Body.Read(buf)
			*/data := string(buf) //[0:num])

			//data := c.PostForm("data")
			//log.Printf("Got data (%s): %s", c.ContentType(), data)
			coI := strings.Index(string(data), ",")
			typ := strings.TrimSuffix(data[5:coI], ";base64")

			file, err := base64.StdEncoding.DecodeString(data[coI+1:])
			if err != nil {
				logError(c, err)
				return
			}
			exts, err := mime.ExtensionsByType(typ)
			if len(exts) == 0 {
				logError(c, err)
				return
			}
			dst, err := ioutil.TempFile(cfg.ImgDir, "*"+exts[0])
			if err != nil {
				logError(c, err)
				return
			}
			defer dst.Close()
			_, err = dst.Write(file)
			if err != nil {
				logError(c, err)
				return
			}
			name = dst.Name()

		}

		// TODO: send preview gen command to goroutine

		n := filepath.Base(name)
		//		c.String(http.StatusOK, fmt.Sprintf("%d files uploaded!", len(files)))
		c.Redirect(http.StatusFound, "/preview/"+n)

	})
	router.GET("/upload", func(c *gin.Context) {
		url := c.Query("url")
		response, err := http.Get(url)
		if err != nil || response.StatusCode != http.StatusOK {
			c.Status(http.StatusServiceUnavailable)
			return
		}

		reader := response.Body
		//contentLength := response.ContentLength
		contentType := response.Header.Get("Content-Type")

		exts, err := mime.ExtensionsByType(contentType)
		if len(exts) == 0 {
			log.Fatal(err)
		}
		dst, err := ioutil.TempFile(cfg.ImgDir, "*"+exts[0])
		if err != nil {
			log.Fatal(err)
		}
		defer dst.Close()
		_, err = io.Copy(dst, reader)
		if err != nil {
			log.Fatal(err)
		}
		n := filepath.Base(dst.Name())
		//		c.String(http.StatusOK, fmt.Sprintf("%d files uploaded!", len(files)))
		c.Redirect(http.StatusFound, "/preview/"+n)
	})

	//router.HEAD("/preview/:name", func(c *gin.Context) {
	router.GET("/preview/:name", func(c *gin.Context) {
		name := c.Param("name")

		// TODO: regexp
		orig := filepath.Join(cfg.ImgDir, name)
		if _, err := os.Stat(orig); err == nil {
			// source image exists
			preview := filepath.Join(cfg.PreviewDir, name)
			if _, err := os.Stat(preview); err == nil {
				// return preview
				c.File(preview)
			} else if os.IsNotExist(err) {
				// TODO: send preview gen request
				// ...
				src, err := imaging.Open(orig)
				if err != nil {
					log.Fatalf("failed to open image: %v", err)
				}

				// Resize srcImage to size = 128x128px using the Lanczos filter.
				dst := imaging.Resize(src, 100, 100, imaging.Lanczos)
				err = imaging.Save(dst, preview)
				if err != nil {
					log.Fatalf("failed to save image: %v", err)
				}

				// return preview
				c.File(preview)
			} else {
				log.Fatal(err)
			}

		} else if os.IsNotExist(err) {
			// return 404

		} else {
			log.Fatal(err)
		}

	})

	return router
}

func logError(c *gin.Context, e error) {
	c.String(http.StatusInternalServerError, "Error: %s", e.Error())
}

/*

 */

func main() {
	cfg, err := setupConfig()
	if err != nil {
		if err.Error() == "ERR1" {
			os.Exit(1)
		}
		os.Exit(2)
	}
	r := setupRouter(cfg)
	r.Run(cfg.Addr)
}
