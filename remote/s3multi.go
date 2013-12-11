package remote

import (
  "github.com/lachie/goamz/s3"
  "dogestry/utils"

  "io"
  "fmt"

  "time"
)

var (
  Megabyte int64 = 1024 * 1024
  S3MinPartSize int64 = 5 * Megabyte
)
type s3request struct {
	method   string
	bucket   string
	path     string
	signpath string
	params   url.Values
	headers  http.Header
	baseurl  string
	payload  io.Reader
	prepared bool
}



type partJob struct {
  multi *s3.Multi
  part *s3.Part
  section *io.SectionReader
  size int64
  md5b64 string
  md5hex string
}
type partResult struct {
  part *s3.Part
  err error
}


func putFileMulti(bucket *s3.Bucket, key string, r io.ReaderAt, totalSize int64, partSize int64, contentType string, acl s3.ACL) error {
  fmt.Println("ok putting", key)

  if partSize < 5 * Megabyte {
    return fmt.Errorf("partSize %s is too small (must be at least 5Mb)\n", utils.HumanSize(partSize))
  }

  m,err := bucket.Multi(key, contentType, acl)
  if err != nil {
    return err
  }

	existing, err := m.ListParts()
	if err != nil && !hasCode(err, "NoSuchUpload") {
		return err
	}

  current := 1

  partc := make(chan partJob, 100)
  resultc := make(chan partResult, 100)

  workerCount := 3
  for w := 0; w < workerCount; w++ {
    go partUploader(w, partc, resultc)
  }


	for offset := int64(0); offset < totalSize; offset += partSize {
    fmt.Printf("offset %d current %d\n", offset, current)

    part := findMultiPart(existing, current)
    current++

    partc <- partJob{
      multi: &m,
      part: &part,
		  section: io.NewSectionReader(r, offset, partSize),
    }
  }
  close(partc)


  partLen := current-1
  uploadedParts := make([]s3.Part, partLen)
  for i := 0; i < partLen; i++ {
    result := <-resultc
    // XXX fail fast?
    if result.err != nil {
      // add extra info
      return result.err
    }

    uploadedParts[result.part.N-1] = result.part
  }

  return nil
}


func partUploader(id int, jobs <-chan partJob, results chan<- partResult) {
  fmt.Printf("worker %d waiting for jobs\n", id)

  for job := range jobs {
    fmt.Printf("uploader %d processing %d\n", id, job.part.N)

    result := partResult{part: &job.part}

    if job.shouldUpload() {
      result.err := job.put()
    }

    results <- result
  }

  fmt.Println("uploader done", id)
}


func (job *partJob) calculate() error {
  if job.md5hex != "" {
    return nil
  }

	_, err = job.section.Seek(0, 0)
	if err != nil {
		return err
	}
	digest := md5.New()
	job.size, err = io.Copy(digest, job.section)
	if err != nil {
		return err
	}

	sum := digest.Sum(nil)
	job.md5hex = hex.EncodeToString(sum)
	job.md5b64 = base64.StdEncoding.EncodeToString(sum)

	return nil
}


func (job *partJob) shouldUpload() bool {
  job.calculate()
  job.md5hex != `"` + job.part.ETag + `"`
}


func (job *partJob) put() (error) {
  m := job.multi
  job.calculate()

	headers := map[string][]string{
		"Content-Length": {strconv.FormatInt(job.size, 10)},
		"Content-MD5":    {job.md5b64},
	}
	params := map[string][]string{
		"uploadId":   {m.UploadId},
		"partNumber": {strconv.FormatInt(int64(job.part.N), 10)},
	}

	for attempt := attempts.Start(); attempt.Next(); {
		_, err := job.section.Seek(0, 0)
		if err != nil {
			return err
		}

		req := &s3request{
			method:  "PUT",
			bucket:  m.Bucket.Name,
			path:    m.Key,
			headers: headers,
			params:  params,
			payload: job.section,
		}

    // ugh
		err = m.Bucket.S3.prepare(req)
		if err != nil {
			return err
		}

		resp, err := m.Bucket.S3.run(req, nil)
		if shouldRetry(err) && attempt.HasNext() {
			continue
		}
		if err != nil {
			return err
		}

		etag := resp.Header.Get("ETag")
		if etag == "" {
			return errors.New("part upload succeeded with no ETag")
		}

    job.part.ETag = etag
    job.part.Size = job.size

		return nil
	}
	panic("unreachable")
}



func findMultiPart(parts []s3.Part, current int) s3.Part {
  for _,part := range parts {
    if part.N == current {
      return part
    }
  }

  return s3.Part{
    N: current,
  }
}


func hasCode(err error, code string) bool {
	s3err, ok := err.(*s3.Error)
	return ok && s3err.Code == code
}
