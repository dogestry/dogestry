package client

import (
	"encoding/json"
	"net/http"

  "fmt"
)


type Container struct {
  ID string
}

type Containers struct {
		ID         string `json:"Id"`
		Image      string
		Command    string
		Created    int64
		Status     string
		Ports      []Port
		SizeRw     int64
		SizeRootFs int64
		Names      []string
	}


type NoSuchContainer struct {
  ID string
}

func (err NoSuchContainer) Error() string {
  return "No such container: " + err.ID
}


type ListContainersOptions struct {
  All    bool
  Size   bool
  Limit  int
  Since  string
  Before string
}


func (c *Client) ListContainers(opts ListContainersOptions) ([]Containers, error) {
  path := "/containers/json?" + queryString(opts)
  body, _, err := c.do("GET", path, nil)
  if err != nil {
    return nil, err
  }

  fmt.Println("conts", string(body))

  var containers []Containers
  err = json.Unmarshal(body, &containers)
  if err != nil {
    return nil, err
  }
  return containers, nil
}


func (c *Client) InspectContainer(id string) (*Container, error) {
  path := "/containers/" + id + "/json"
  body, status, err := c.do("GET", path, nil)
  if status == http.StatusNotFound {
    return nil, &NoSuchContainer{ID: id}
  }
  if err != nil {
    return nil, err
  }

  fmt.Println("cont", id, body)

  var container Container
  err = json.Unmarshal(body, &container)
  if err != nil {
    return nil, err
  }
  return &container, nil
}
