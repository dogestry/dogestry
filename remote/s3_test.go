package remote

import (
	"testing"
	"time"

	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"github.com/crowdmob/goamz/testutil"
	"github.com/dogestry/dogestry/config"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type S struct {
	remote  *S3Remote
	TempDir string
}

var _ = Suite(&S{})

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *C) {
	testServer.Start()

	auth, _ := aws.GetAuth("abc", "123", "", time.Time{})
	client := s3.New(auth, aws.Region{Name: "faux-region-1", S3Endpoint: testServer.URL})

	tempDir, err := ioutil.TempDir("", "dogestry-test")
	if err != nil {
		c.Fatalf("couldn't get tempdir. Error: %s", err)
	}

	s.TempDir = tempDir

	baseConfig, err := config.NewConfig(false, 22375, false, false)
	if err != nil {
		c.Fatalf("couldn't initialize config. Error: %s", err)
	}

	s.remote = &S3Remote{
		config:     baseConfig,
		BucketName: "bucket",
		client:     client,
	}
}

func (s *S) TearDownSuite(c *C) {
	defer os.RemoveAll(s.TempDir)
}

func (s *S) TestBucket(c *C) {
	testServer.Response(200, nil, "content")

	b := s.remote.getBucket()
	c.Assert(b.Name, Equals, "bucket")
}

func (s *S) TestRepoKeys(c *C) {
	nelsonSha := "123"

	//testServer.Response(200, nil, "content")
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, nelsonSha)

	keys, err := s.remote.repoKeys("")
	c.Assert(err, IsNil)

	testServer.WaitRequests(2)

	c.Log(keys["Nelson"])

	c.Assert(keys["Nelson"].key, Equals, "Nelson")
	c.Assert(keys["Nelson"].Sum(), Equals, nelsonSha)

	c.Assert(keys["Neo"].key, Equals, "Neo")
	c.Assert(keys["Neo"].Sum(), Equals, "")
}

func (s *S) TestLocalKeys(c *C) {
	dumpFile(s.TempDir, "file1", "hello world")
	dumpFile(s.TempDir, "dir/file2", "hello mars")

	keys, err := s.remote.localKeys(s.TempDir)
	c.Assert(err, IsNil)

	c.Assert(keys["file1"].key, Equals, "file1")
	c.Assert(keys["file1"].fullPath, Equals, filepath.Join(s.TempDir, "file1"))
	c.Assert(keys["file1"].sum, Equals, "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed")

	c.Assert(keys["dir/file2"].key, Equals, "dir/file2")
	c.Assert(keys["dir/file2"].fullPath, Equals, filepath.Join(s.TempDir, "dir/file2"))
	c.Assert(keys["dir/file2"].sum, Equals, "dd6944c43fabd03cf643fe0daf625759dbdea808")
}

func (s *S) TestResolveImageNameToId(c *C) {
	rubyId := "123"

	testServer.Response(200, nil, "123")

	id, err := s.remote.ResolveImageNameToId("ruby")
	c.Assert(err, IsNil)

	c.Assert(string(id), Equals, rubyId)

	testServer.Flush()
	testServer.Response(404, nil, "")

	id, err = s.remote.ResolveImageNameToId("rubyx")
	c.Assert(err, Not(IsNil))
}

func dumpFile(temp, filename, content string) error {
	out := filepath.Join(temp, filename)
	if err := os.MkdirAll(filepath.Dir(out), 0700); err != nil {
		return err
	}
	return ioutil.WriteFile(out, []byte(content), 0600)
}
