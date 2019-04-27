package ginupload

import (
	"github.com/gin-gonic/gin"
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
		HandleMultiPartFunc: func(form *multipart.Form) (*string, error) { n := "/index"; return &n, nil },
		HandleBase64Func:    func(data, name string) (*string, error) { n := "/" + name; return &n, nil },
		HandleURLFunc:       func(url string) (*string, error) { n := "/" + path.Base(url); return &n, nil },
	})
}

func (ss *ServerSuite) TestHandleMultiPart() {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request, _ = http.NewRequest("POST", "/upload", strings.NewReader(`fake data`))
	c.Request.Header.Set("Content-Type", "multipart/form-data")
	ss.srv.HandleMultiPart(c)
	assert.Equal(ss.T(), http.StatusInternalServerError, resp.Code)
}

func (ss *ServerSuite) TestHandleBase64() {
	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request, _ = http.NewRequest("POST", "/upload", strings.NewReader(`{"data":"data:image/png;base64,iVBORw0K","name":"file.ext"}`))
	ss.srv.HandleBase64(c)
	assert.Equal(ss.T(), http.StatusOK, resp.Code)
	assert.Equal(ss.T(), `{"file":"/img/file.ext","preview":"/preview/file.ext"}`, resp.Body.String())
}

func (ss *ServerSuite) TestHandleURL() {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("body"))
	}))
	defer func() { testServer.Close() }()

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request, _ = http.NewRequest(http.MethodGet, "/upload?url="+testServer.URL, nil)
	ss.srv.HandleURL(c)
	assert.Equal(ss.T(), http.StatusFound, resp.Code)
}

func TestSuite(t *testing.T) {
	myTest := &ServerSuite{}
	suite.Run(t, myTest)
}
