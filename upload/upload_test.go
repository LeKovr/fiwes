package upload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	mapper "github.com/birkirb/loggers-mapper-logrus"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/udhos/equalfile"
)

const (
	badBase64       = "data:image/png;base64,тест"
	badBase64Image  = "data:image/png;base64,iVBORw0K"
	badBase64Prefix = "data:image/pn;base64,iVBORw0K"
)

type ServerSuite struct {
	suite.Suite
	cfg  Config
	srv  *Service
	hook *test.Hook
	root string
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
	ss.root, err = os.MkdirTemp("", "img")
	require.NoError(ss.T(), err)
	ss.cfg.Dir = filepath.Join(ss.root, "/img")
	ss.cfg.PreviewDir = filepath.Join(ss.root, "/preview")
	ss.cfg.AllowedImageHosts = []string{"127.0.0.1"}
	ss.srv = New(ss.cfg, log)
}

func (ss *ServerSuite) TearDownSuite() {
	os.RemoveAll(ss.root)
}

func (ss *ServerSuite) TestHandleMultiPart() {
	path := filepath.Join("../testdata", "pic.jpg")
	f, err := os.Open(path)
	require.NoError(ss.T(), err)
	defer f.Close()
	fileContents, err := io.ReadAll(f)
	require.NoError(ss.T(), err)

	tests := []struct {
		name  string
		field string
		err   error
	}{
		{"OK", "file", nil},
		{"NoFile", "file1", errors.New(ErrNoSingleFile)},
	}
	for _, tt := range tests {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile(tt.field, "file.jpg")
		require.NoError(ss.T(), err)
		_, err = part.Write(fileContents)
		require.NoError(ss.T(), err)
		err = writer.Close()
		require.NoError(ss.T(), err)
		req, _ := http.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		err = req.ParseMultipartForm(32 << 20)
		require.NoError(ss.T(), err)
		_, err = ss.srv.HandleMultiPart(req.MultipartForm)
		if tt.err != nil {
			require.NotNil(ss.T(), err, tt.name)
			assert.Equal(ss.T(), tt.err.Error(), err.Error(), tt.name)
			continue
		}
		require.NoError(ss.T(), err, tt.name)
		name, err := ss.srv.HandleMultiPart(req.MultipartForm)
		require.NoError(ss.T(), err, tt.name)
		cmp := equalfile.New(nil, equalfile.Options{}) // compare using single mode
		equal, err := cmp.CompareFile("../testdata/pic100.jpg", ss.root+"/preview"+*name)
		require.NoError(ss.T(), err, tt.name)
		assert.True(ss.T(), equal, tt.name)
	}
}

func (ss *ServerSuite) TestHandleBase64BadRequest() {
	ss.hook.Reset()
	_, err := ss.srv.HandleBase64(badBase64, "file.png")
	ss.printLogs()
	require.NotNil(ss.T(), err)
	httpErr, ok := err.(interface{ Status() int })
	assert.True(ss.T(), ok)
	assert.Equal(ss.T(), http.StatusBadRequest, httpErr.Status())
}

func (ss *ServerSuite) TestHandleBase64BadPrefix() {
	ss.hook.Reset()
	_, err := ss.srv.HandleBase64(badBase64Prefix, "file.png")
	ss.printLogs()
	require.NotNil(ss.T(), err)
	httpErr, ok := err.(interface{ Status() int })
	assert.True(ss.T(), ok)
	assert.Equal(ss.T(), http.StatusBadRequest, httpErr.Status())
}

// File hold JSON request struct
type File struct {
	Name string `form:"name" json:"name" binding:"required"`
	Data string `form:"data" json:"data" binding:"required"`
}

func (ss *ServerSuite) TestHandleBase64OK() {
	ss.hook.Reset()
	js := &File{}
	helperLoadJSON(ss.T(), "build", js)
	name, err := ss.srv.HandleBase64(js.Data, js.Name)
	require.NoError(ss.T(), err)
	cmp := equalfile.New(nil, equalfile.Options{}) // compare using single mode
	equal, err := cmp.CompareFile("../testdata/build100.png", ss.root+"/preview"+*name)
	require.NoError(ss.T(), err)
	assert.True(ss.T(), equal)
}

func (ss *ServerSuite) TestHandleBase64NoExt() {
	ss.hook.Reset()
	js := &File{}
	helperLoadJSON(ss.T(), "build", js)
	_, err := ss.srv.HandleBase64(js.Data, "build")
	require.EqualError(ss.T(), err, ErrBadFilename)

	name, err := ss.srv.HandleBase64(js.Data, "build.png")
	require.NoError(ss.T(), err)

	cmp := equalfile.New(nil, equalfile.Options{}) // compare using single mode
	equal, err := cmp.CompareFile("../testdata/build100.png", ss.root+"/preview"+*name)
	require.NoError(ss.T(), err)
	assert.True(ss.T(), equal)

	// test same name with error
	_, err = ss.srv.HandleBase64(badBase64Image, "build")
	require.NotNil(ss.T(), err)
	httpErr, ok := err.(interface{ Status() int })
	assert.True(ss.T(), ok)
	assert.Equal(ss.T(), http.StatusBadRequest, httpErr.Status())

}

func (ss *ServerSuite) TestHandleBase64BadMedia() {
	ss.hook.Reset()
	js := &File{}
	helperLoadJSON(ss.T(), "unknown", js)
	_, err := ss.srv.HandleBase64(js.Data, js.Name)
	require.NotNil(ss.T(), err)
	httpErr, ok := err.(interface{ Status() int })
	assert.True(ss.T(), ok)
	assert.Equal(ss.T(), http.StatusBadRequest, httpErr.Status())
}

func (ss *ServerSuite) TestHandleBase64NoExtRandom() {
	ss.hook.Reset()
	js := &File{}
	helperLoadJSON(ss.T(), "build", js)
	ss.srv.Config.UseRandomName = true // TODO: This is incompartible with parallel tests
	name, err := ss.srv.HandleBase64(js.Data, "build.png")
	ss.srv.Config.UseRandomName = false // TODO: This is incompartible with parallel tests
	require.NoError(ss.T(), err)
	cmp := equalfile.New(nil, equalfile.Options{}) // compare using single mode
	equal, err := cmp.CompareFile("../testdata/build100.png", ss.root+"/preview"+*name)
	require.NoError(ss.T(), err)
	assert.True(ss.T(), equal)
}

func (ss *ServerSuite) TestHandleURLOK() {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.ServeFile(res, req, "../testdata/build.png")
	}))
	defer func() { testServer.Close() }()

	name, err := ss.srv.HandleURL(testServer.URL + "/build.png")

	require.NoError(ss.T(), err)
	cmp := equalfile.New(nil, equalfile.Options{}) // compare using single mode
	equal, err := cmp.CompareFile("../testdata/build100.png", ss.root+"/preview"+*name)
	require.NoError(ss.T(), err)
	assert.True(ss.T(), equal)
}

func (ss *ServerSuite) TestHandleURLNotFound() {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusNotFound)
	}))
	defer func() { testServer.Close() }()

	_, err := ss.srv.HandleURL(testServer.URL + "/build.png")
	require.NotNil(ss.T(), err)
	assert.Equal(ss.T(), fmt.Sprintf(ErrFmtBadDownload, 404), err.Error())
}

func (ss *ServerSuite) TestHandleURLNotAvailable() {
	_, err := ss.srv.HandleURL("http://127.0.0.1:1/build.png")
	require.NotNil(ss.T(), err)

	httpErr, ok := err.(interface{ Status() int })
	assert.True(ss.T(), ok)
	assert.Equal(ss.T(), http.StatusServiceUnavailable, httpErr.Status())
}

func TestSuite(t *testing.T) {
	myTest := &ServerSuite{}
	suite.Run(t, myTest)
}

func (ss *ServerSuite) printLogs() {
	for _, e := range ss.hook.Entries {
		fmt.Printf("ENT[%s]: %s\n", e.Level, e.Message)
	}
}

func helperLoadJSON(t *testing.T, name string, data interface{}) {
	path := filepath.Join("../testdata", name+".json") // relative path
	bytes, err := os.ReadFile(path)
	require.NoError(t, err)
	err = json.Unmarshal(bytes, &data)
	require.NoError(t, err)
}
