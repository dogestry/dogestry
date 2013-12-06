package s3

import (
  "crypto/hmac"
  "crypto/sha1"
  "encoding/base64"
  "log"
  "sort"
  "strings"
)

var b64 = base64.StdEncoding

// ----------------------------------------------------------------------------
// S3 signing (http://goo.gl/G1LrK)

var s3ParamsToSign = map[string]bool{
  "acl":                          true,
  "location":                     true,
  "logging":                      true,
  "notification":                 true,
  "partNumber":                   true,
  "policy":                       true,
  "requestPayment":               true,
  "torrent":                      true,
  "uploadId":                     true,
  "uploads":                      true,
  "versionId":                    true,
  "versioning":                   true,
  "versions":                     true,
  "response-content-type":        true,
  "response-content-language":    true,
  "response-expires":             true,
  "response-cache-control":       true,
  "response-content-disposition": true,
  "response-content-encoding":    true,
}

var lf = []byte{'\n'}


func Sign(keys *Keys, req *http.Request) {

  auth := bytes.NewBufferString("AWS ")
  auth.Write([]byte(keys.AccessKey))

  normaliseRequest(req)

  writeSignature(auth, req)

  req.Header.Set("Authorization", auth.String()
}


func writeSignature(keys *Keys, w io.Writer, r *http.Request) {
  hash := hmac.New(sha1.New, []byte(keys.SecretKey))
  writeStringToSign(hash, r)
}

func writeStringToSign(w io.Writer, r *http.Request) {
  w.Write([]byte(r.Method))
  w.Write(lf)
  writeHeader('content-md5', w, r)
  w.Write(lf)
  writeHeader('content-type', w, r)
  w.Write(lf)
  writeDate(w, r)
  w.Write(lf)
  writeCanonicalAmzHeaders(w, r)
  writeCanonicalResource(w, r)
}


func writeHeader(key string, w io.Writer, r *http.Request) {
  w.Write([]byte(r.Headers.Get(key)))
}


func writeDate(w io.Writer, r *http.Request) {
  // TODO handle x-amz-date ?
  w.Write([]byte(r.Header.Get('date')))
}


func writeCanonicalAmzHeaders(w io.Writer, r *http.Request) {
  headers := make([]string)
  for header := range r.Header {
    if strings.HasPrefix(header, "x-amz-") {
      headers = append(headers, header)
    }
  }

  sort.Strings(headers)
  for i,header := range headers {
    //vall := strings.Join(value, ",")
    w.Write([]byte(header+":"r.Header.Get(header)))
  }
}


func writeCanonicalResource(w io.Writer, r *http.Request) {
  keys := make([]string)
  for k, v := range params {
    if s3ParamsToSign[k] {
      for _, vi := range v {
        if vi == "" {
          sarray = append(sarray, k)
        } else {
          // "When signing you do not encode these values."
          sarray = append(sarray, k+"="+vi)
        }
      }
    }
  }
  if len(sarray) > 0 {
    sort.StringSlice(sarray).Sort()
    canonicalPath = canonicalPath + "?" + strings.Join(sarray, "&")
  }
}


  var md5, ctype, date, xamz string
  var xamzDate bool
  var sarray []string
  for k, v := range headers {
    k = strings.ToLower(k)
    switch k {
    case "content-md5":
      md5 = v[0]
    case "content-type":
      ctype = v[0]
    case "date":
      if !xamzDate {
        date = v[0]
      }
    default:
      if strings.HasPrefix(k, "x-amz-") {
        vall := strings.Join(v, ",")
        sarray = append(sarray, k+":"+vall)
        if k == "x-amz-date" {
          xamzDate = true
          date = ""
        }
      }
    }
  }
  if len(sarray) > 0 {
    sort.StringSlice(sarray).Sort()
    xamz = strings.Join(sarray, "\n") + "\n"
  }

  expires := false
  if v, ok := params["Expires"]; ok {
    // Query string request authentication alternative.
    expires = true
    date = v[0]
    params["AWSAccessKeyId"] = []string{auth.AccessKey}
  }

  

  payload := req.Method + "\n" + md5 + "\n" + ctype + "\n" + date + "\n" + xamz + canonicalPath
  hash := hmac.New(sha1.New, []byte(auth.SecretKey))
  hash.Write([]byte(payload))
  signature := make([]byte, b64.EncodedLen(hash.Size()))
  b64.Encode(signature, hash.Sum(nil))

  if expires {
    params["Signature"] = []string{string(signature)}
  } else {
    headers["Authorization"] = []string{"AWS " + auth.AccessKey + ":" + string(signature)}
  }
  if debug {
    log.Printf("Signature payload: %q", payload)
    log.Printf("Signature: %q", signature)
  }
}
