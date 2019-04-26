package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mapper "github.com/birkirb/loggers-mapper-logrus"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"imgserv/ginupload"
)

func TestSetupConfig(t *testing.T) {
	cfg, err := setupConfig("--http_addr", "80")
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	_, err = setupConfig("-h")
	assert.Equal(t, ErrGotHelp, err)

	_, err = setupConfig("--unknown")
	assert.NotNil(t, err)
	assert.Equal(t, "unknown flag `unknown'", err.Error())

}
func TestSetupLog(t *testing.T) {
	l := setupLog()
	assert.NotNil(t, l)
}

func TestBase64(t *testing.T) {
	cfg := &Config{Img: ginupload.Config{Path: "/img", PreviewPath: "/preview", UploadPath: "/upload"}}

	l, hook := test.NewNullLogger()
	//ss.hook = hook
	l.SetLevel(logrus.DebugLevel)
	log := mapper.NewLogger(l)
	hook.Reset()

	router := setupRouter(cfg, log)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", strings.NewReader("data:image/png;base64,iVBORw0K"))
	router.ServeHTTP(w, req)

	assert.Equal(t, 415, w.Code)
	assert.Equal(t, "Unsupported media type", w.Body.String())
}

func TestExternal(t *testing.T) {
	cfg := &Config{Img: ginupload.Config{Path: "/img", PreviewPath: "/preview", UploadPath: "/upload"}}

	l, hook := test.NewNullLogger()
	//ss.hook = hook
	l.SetLevel(logrus.DebugLevel)
	log := mapper.NewLogger(l)
	hook.Reset()

	router := setupRouter(cfg, log)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/upload?url=/img/xx.png", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "Error: Get /img/xx.png: unsupported protocol scheme \"\"", w.Body.String())
}
