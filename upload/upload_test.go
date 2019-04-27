package upload

import (
	//	"net/http"
	//	"net/http/httptest"
	//	"bytes"
	"fmt"
	"io/ioutil"
	"os"
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
	dir, err := ioutil.TempDir("", "img")
	if err != nil {
		log.Fatal(err)
	}
	ss.cfg.Dir = dir
	ss.srv = New(ss.cfg, log)
}

func (ss *ServerSuite) TearDownSite() {
	os.RemoveAll(ss.cfg.Dir)
}

func (ss *ServerSuite) TestHandleBase64() {
	ss.hook.Reset()
	_, err := ss.srv.HandleBase64("data:image/png;base64,iVBORw0K", "file.png")
	ss.printLogs()
	assert.Equal(ss.T(), ErrNotImage, err)
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
