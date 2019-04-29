package ginupload

import (
	"bytes"
	"errors"
	"github.com/gin-gonic/gin"
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

	"fiwes/upload"
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
				return nil, errors.New(upload.ErrNoSingleFile)
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
				return nil, errors.New("download failed:" + url)
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

func (ss *ServerSuite) TestHandleMultiPartNoForm() {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request, _ = http.NewRequest("POST", "/upload", strings.NewReader(`fake data`))
	c.Request.Header.Set("Content-Type", "multipart/form-data")
	ss.srv.HandleMultiPart(c)
	assert.Equal(ss.T(), http.StatusInternalServerError, resp.Code)
}

func (ss *ServerSuite) TestHandleMultiPartNoFile() {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file1", "file.png")
	require.NoError(ss.T(), err)
	_, err = part.Write([]byte{})
	require.NoError(ss.T(), err)
	err = writer.Close()
	require.NoError(ss.T(), err)

	c.Request, _ = http.NewRequest("POST", "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	ss.srv.HandleMultiPart(c)
	assert.Equal(ss.T(), http.StatusInternalServerError, resp.Code)

}

func (ss *ServerSuite) TestHandleMultiPart() {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "file.png")
	require.NoError(ss.T(), err)
	_, err = part.Write([]byte{})
	require.NoError(ss.T(), err)
	err = writer.Close()
	require.NoError(ss.T(), err)

	c.Request, _ = http.NewRequest("POST", "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	ss.srv.HandleMultiPart(c)
	assert.Equal(ss.T(), http.StatusOK, resp.Code)

}

func (ss *ServerSuite) TestHandleBase64() {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request, _ = http.NewRequest("POST", "/upload", strings.NewReader(`{"data":"data:image/png;base64,iVBORw0K","name":"file.png"}`))
	ss.srv.HandleBase64(c)
	assert.Equal(ss.T(), http.StatusOK, resp.Code)
	assert.Equal(ss.T(), `{"file":"/img/file.png","preview":"/preview/file.png"}`, resp.Body.String())
}
func (ss *ServerSuite) TestHandleBase64NoImage() {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request, _ = http.NewRequest("POST", "/upload", strings.NewReader(`{"data":"data:image/png;base64,iVBORw0K","name":"file.ext"}`))
	ss.srv.HandleBase64(c)
	assert.Equal(ss.T(), http.StatusUnsupportedMediaType, resp.Code)
}

func (ss *ServerSuite) TestHandleBase64NoJSON() {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request, _ = http.NewRequest("POST", "/upload", strings.NewReader(``))
	ss.srv.HandleBase64(c)
	assert.Equal(ss.T(), http.StatusBadRequest, resp.Code)
}

func (ss *ServerSuite) TestHandleURLError() {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, err := res.Write([]byte("body"))
		if err != nil {
			log.Fatal(err)
		}
	}))
	defer func() { testServer.Close() }()

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	//	c.Request, _ = http.NewRequest(http.MethodGet, "/upload?url="+testServer.URL+"/file.png", nil)
	c.Request, _ = http.NewRequest(http.MethodGet, "/upload?url="+testServer.URL+"/error.png", nil)
	ss.srv.HandleURL(c)

	//	assert.Equal(ss.T(), http.StatusFound, resp.Code)
	//	assert.Equal(ss.T(), "<a href=\"/preview/file.png\">Found</a>.\n\n", resp.Body.String())
	assert.Equal(ss.T(), http.StatusInternalServerError, resp.Code)
}

func (ss *ServerSuite) TestHandleURL() {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, err := res.Write([]byte("body"))
		if err != nil {
			log.Fatal(err)
		}
	}))
	defer func() { testServer.Close() }()

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request, _ = http.NewRequest(http.MethodGet, "/upload?url="+testServer.URL+"/file.png", nil)
	//c.Request, _ = http.NewRequest(http.MethodGet, "/upload?url="+testServer.URL+"/error.png", nil)
	ss.srv.HandleURL(c)

	assert.Equal(ss.T(), http.StatusFound, resp.Code)
	assert.Equal(ss.T(), "<a href=\"/preview/file.png\">Found</a>.\n\n", resp.Body.String())
	//assert.Equal(ss.T(), http.StatusInternalServerError, resp.Code)
}

func TestSuite(t *testing.T) {
	myTest := &ServerSuite{}
	suite.Run(t, myTest)
}
