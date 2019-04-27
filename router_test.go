package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mapper "github.com/birkirb/loggers-mapper-logrus"
	"github.com/gin-gonic/gin"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSetupConfig(t *testing.T) {
	cfg, err := setupConfig("--http_addr", "80")
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	_, err = setupConfig("-h")
	assert.Equal(t, ErrGotHelp, err)

	_, err = setupConfig()
	assert.Equal(t, "unknown flag `t'", err.Error())

	_, err = setupConfig("--unknown")
	assert.NotNil(t, err)
	assert.Equal(t, "unknown flag `unknown'", err.Error())

}

func TestSetupLog(t *testing.T) {
	mode := gin.Mode()
	gin.SetMode(gin.DebugMode)
	l := setupLog()
	if mode != gin.DebugMode {
		gin.SetMode(mode)
	}
	assert.NotNil(t, l)
}

type ServerSuite struct {
	suite.Suite
	cfg  *Config
	srv  *gin.Engine
	hook *test.Hook
}

func (ss *ServerSuite) SetupSuite() {

	// Fill config with default values
	ss.cfg = &Config{}
	p := flags.NewParser(ss.cfg, flags.Default)
	_, err := p.ParseArgs([]string{"--html"})
	require.NoError(ss.T(), err)

	l, hook := test.NewNullLogger()
	ss.hook = hook
	l.SetLevel(logrus.DebugLevel)
	log := mapper.NewLogger(l)
	hook.Reset()
	ss.srv = setupRouter(ss.cfg, log)
}

func TestSuite(t *testing.T) {
	myTest := &ServerSuite{}
	suite.Run(t, myTest)
}

func (ss *ServerSuite) TestMultiPart() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader(`fake data`))
	req.Header.Set("Content-Type", "multipart/form-data")
	ss.srv.ServeHTTP(w, req)

	assert.Equal(ss.T(), http.StatusInternalServerError, w.Code)
	assert.Equal(ss.T(), "Error: no multipart boundary param in Content-Type", w.Body.String())
}

// TODO: remove created dir
func (ss *ServerSuite) TestBase64() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader(`{"data":"data:image/png;base64,iVBORw0K","name":"file.ext"}`))
	ss.srv.ServeHTTP(w, req)

	assert.Equal(ss.T(), 415, w.Code)
	assert.Equal(ss.T(), "Unsupported media type", w.Body.String())
}

func (ss *ServerSuite) TestURL() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/upload?url=/img/xx.png", nil)
	ss.srv.ServeHTTP(w, req)

	assert.Equal(ss.T(), 500, w.Code)
	assert.Equal(ss.T(), "Error: Get /img/xx.png: unsupported protocol scheme \"\"", w.Body.String())
}
