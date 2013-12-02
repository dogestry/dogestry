package client

import (
  "io"
)

func (c *Client) GetImageTarball(imageName string, w io.Writer) error {
  return c.stream("GET", "/images/"+imageName+"/get", nil, w)
}
