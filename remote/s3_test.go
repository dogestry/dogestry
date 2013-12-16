package remote

import (
	//"bytes"
	//"io/ioutil"
	//"net/http"
	//"strings"

	"testing"
	"time"

	"github.com/lachie/goamz/aws"
	"github.com/lachie/goamz/s3"
	"github.com/lachie/goamz/testutil"
	. "launchpad.net/gocheck"

  "dogestry/config"
)

func Test(t *testing.T) {
	TestingT(t)
}

type S struct {
	remote *S3Remote
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

  //remote,err := NewRemote("s3://bucket/prefix&region=faux-region-1", baseConfig)
  //if err != nil {
    //panic(err)
  //}

  //s.remote = remote.(*S3Remote)


  s.remote = &S3Remote{
    config: baseConfig,
    BucketName: "bucket",
    KeyPrefix:  "prefix",
    client:     client,
  }
}

func (s *S) TearDownSuite(c *C) {
  s3.SetAttemptStrategy(nil)
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
