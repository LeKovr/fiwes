package ginupload

import (
	"bytes"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	mapper "github.com/birkirb/loggers-mapper-logrus"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/LeKovr/fiwes/upload"
)

type ServerSuite struct {
	suite.Suite
	cfg  Config
	srv  *Service
	hook *test.Hook
}

func (ss *ServerSuite) SetupSuite() {

	// Fill config with default values
	p := flags.NewParser(&ss.cfg, flags.Default)
	_, err := p.ParseArgs([]string{})
	require.NoError(ss.T(), err)

	l, hook := test.NewNullLogger()
	ss.hook = hook
	l.SetLevel(logrus.DebugLevel)
	log := mapper.NewLogger(l)

	hook.Reset()

	ss.srv = New(ss.cfg, log, &UploaderMock{
		HandleMultiPartFunc: func(form *multipart.Form) (*string, error) {
			_, ok := form.File["file"]
			if !ok {
				return nil, upload.NewHTTPError(
					http.StatusBadRequest,
					errors.New(upload.ErrNoSingleFile),
				)
			}
			n := "/index"
			return &n, nil
		},
		HandleBase64Func: func(data, name string) (*string, error) {
			if name != "file.png" {
				return nil, upload.NewHTTPError(http.StatusUnsupportedMediaType, errors.New(upload.ErrNoCTypeExt))
			}
			n := "/" + name
			return &n, nil
		},
		HandleURLFunc: func(url string) (*string, error) {
			if strings.HasSuffix(url, "error.png") {
				return nil, errors.New("unhandled error")
			}
			n := "/" + path.Base(url)
			return &n, nil
		},
	})
}

func (ss *ServerSuite) TestNew() {
	l, _ := test.NewNullLogger()
	log := mapper.NewLogger(l)
	srv := New(ss.cfg, log, nil)
	require.NotNil(ss.T(), srv)
	require.NotNil(ss.T(), srv.up)
}

func (ss *ServerSuite) TestHandleMultiPart() {
	tests := []struct {
		name   string
		reader io.Reader
		field  string
		code   int
	}{
		{"OK", nil, "file", http.StatusOK},
		{"NoFile", nil, "file1", http.StatusBadRequest},
		{"NoForm", strings.NewReader(`fake data`), "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		ctype := "multipart/form-data"
		if tt.reader == nil {
			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile(tt.field, "file.png")
			require.NoError(ss.T(), err)
			_, err = part.Write([]byte{})
			require.NoError(ss.T(), err)
			err = writer.Close()
			require.NoError(ss.T(), err)
			tt.reader = body
			ctype = writer.FormDataContentType()
		}
		c.Request, _ = http.NewRequest("POST", "/upload", tt.reader)
		c.Request.Header.Set("Content-Type", ctype)
		ss.srv.HandleMultiPart(c)
		assert.Equal(ss.T(), tt.code, resp.Code, tt.name)
	}
}

func (ss *ServerSuite) TestHandleBase64() {
	tests := []struct {
		name    string
		reader  io.Reader
		code    int
		message string
	}{
		{"OK", strings.NewReader(`{"data":"data:image/png;base64,iVBORw0K","name":"file.png"}`),
			http.StatusOK, `{"file":"/img/file.png","preview":"/preview/file.png"}`},
		{"NoImage", strings.NewReader(`{"data":"data:image/png;base64,iVBORw0K","name":"file.ext"}`), http.StatusUnsupportedMediaType, ""},
		{"NoJSON", strings.NewReader(``), http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request, _ = http.NewRequest("POST", "/upload", tt.reader)
		ss.srv.HandleBase64(c)
		assert.Equal(ss.T(), tt.code, resp.Code, tt.name)
		if tt.message != "" {
			assert.Equal(ss.T(), tt.message, resp.Body.String(), tt.name)
		}
	}
}

func (ss *ServerSuite) TestHandleURL() {
	tests := []struct {
		name    string
		file    string
		code    int
		message string
	}{
		{"OK", "/file.png", http.StatusFound, "<a href=\"/preview/file.png\">Found</a>.\n\n"},
		{"Error", "/error.png", http.StatusInternalServerError, "unhandled error"},
	}
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, err := res.Write([]byte("body"))
		if err != nil {
			log.Fatal(err)
		}
	}))
	defer func() { testServer.Close() }()

	for _, tt := range tests {
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request, _ = http.NewRequest(http.MethodGet, "/upload?url="+testServer.URL+tt.file, nil)
		ss.srv.HandleURL(c)
		assert.Equal(ss.T(), tt.code, resp.Code, tt.name)
		assert.Equal(ss.T(), tt.message, resp.Body.String(), tt.name)
	}
}

func TestSuite(t *testing.T) {
	myTest := &ServerSuite{}
	suite.Run(t, myTest)
}
