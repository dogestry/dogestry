package remote

import (
	//"bytes"
	//"io/ioutil"
	//"net/http"
	//"strings"

	"testing"
	"time"

  "os"
  "io/ioutil"
  "path/filepath"

	"github.com/lachie/goamz/aws"
	"github.com/lachie/goamz/s3"
	"github.com/lachie/goamz/testutil"
	. "launchpad.net/gocheck"

  "github.com/blake-education/dogestry/config"
)

func Test(t *testing.T) {
	TestingT(t)
}

type S struct {
	remote *S3Remote
  tempDir string
}

var _ = Suite(&S{})

var testServer = testutil.NewHTTPServer()

var baseConfig = RemoteConfig{
  Config: config.Config{
    S3: config.S3Config{
      Access_Key_Id: "abc",
      Secret_Key: "123",
    },
  },
}

func (s *S) SetUpSuite(c *C) {
	testServer.Start()

  auth := aws.Auth{"abc", "123", ""}
  client := s3.New(auth, aws.Region{Name: "faux-region-1", S3Endpoint: testServer.URL})


  tempDir, err := ioutil.TempDir("", "dogestry-test")
  if err != nil {
    c.Fatalf("couldn't get tempdir: %s", err)
  }

  s.tempDir = tempDir

  s.remote = &S3Remote{
    config: baseConfig,
    BucketName: "bucket",
    KeyPrefix:  "prefix",
    client:     client,
  }
}

func (s *S) TearDownSuite(c *C) {
  s3.SetAttemptStrategy(nil)

  defer os.RemoveAll(s.tempDir)
}

func (s *S) SetUpTest(c *C) {
  attempts := aws.AttemptStrategy{
    Total: 300 * time.Millisecond,
    Delay: 100 * time.Millisecond,
  }
  s3.SetAttemptStrategy(&attempts)
}


func (s *S) TestBucket(c *C) {
  testServer.Response(200, nil, "content")

	b := s.remote.getBucket()
  c.Assert(b.Name, Equals, "bucket")
}

func (s *S) TestRepoKeys(c *C) {
  nelsonSha := "123"

  testServer.Response(200, nil, "content")
  testServer.Response(200, nil, GetListResultDump1)
  testServer.Response(200, nil, nelsonSha)

	keys,err := s.remote.repoKeys("")
	c.Assert(err, IsNil)

	testServer.WaitRequest()

  c.Assert(keys["Nelson"].key, Equals, "Nelson")
  c.Assert(keys["Nelson"].sum, Equals, nelsonSha)

  c.Assert(keys["Neo"].key, Equals, "Neo")
  c.Assert(keys["Neo"].sum, Equals, "")
}


func (s *S) TestLocalKeys(c *C) {
  dumpFile(s.tempDir, "file1", "hello world")
  dumpFile(s.tempDir, "dir/file2", "hello mars")

  keys,err := s.remote.localKeys(s.tempDir)
	c.Assert(err, IsNil)

  c.Assert(keys["file1"].key, Equals, "file1")
  c.Assert(keys["file1"].fullPath, Equals, filepath.Join(s.tempDir,"file1"))
  c.Assert(keys["file1"].sum, Equals, "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed")

  c.Assert(keys["dir/file2"].key, Equals, "dir/file2")
  c.Assert(keys["dir/file2"].fullPath, Equals, filepath.Join(s.tempDir,"dir/file2"))
  c.Assert(keys["dir/file2"].sum, Equals, "dd6944c43fabd03cf643fe0daf625759dbdea808")
}


func dumpFile(temp, filename, content string) error {
  out := filepath.Join(temp, filename)
  if err := os.MkdirAll(filepath.Dir(out), 0700); err != nil {
    return err
  }
  return ioutil.WriteFile(out, []byte(content), 0600)
}
