package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/carbocation/interpose"
	"github.com/carbocation/interpose/middleware"
	"github.com/gorilla/mux"
)

var dogeCli *DogestryCli

//
func errJson(msg string) string {
	problem := struct {
		Err string `json:"error"`
	}{
		Err: msg,
	}

	// This is how we generate errors. If an error happens here, well...
	bytes, _ := json.Marshal(problem)
	return string(bytes)
}

func pullHandler(response http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	response.Header().Set("Content-Type", "application/json")

	image := req.URL.Query().Get("fromImage")

	response.Write([]byte(fmt.Sprintf(`{"status": "pulling %s"}`, image)))
	dogeCli.RunCmd("pull", "s3://nr-docker-images/", image)
	response.Write([]byte(`{"status": "done."}`))
}

func healthCheckHandler(response http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	response.Write([]byte("OK"))
}

func rootHandler(response http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	response.Write([]byte(`{"error": "Dogestry API, nothing to see here..."}`))
}

func ServeHttp(address string, cli *DogestryCli) {
	router := mux.NewRouter()
	middle := interpose.New()
	middle.Use(middleware.GorillaLog())
	middle.UseHandler(router)

	dogeCli = cli

	router.Handle("/{version}/images/create", http.HandlerFunc(pullHandler)).Methods("POST")
	router.Handle("/status/check", http.HandlerFunc(healthCheckHandler)).Methods("GET")
	router.Handle("/", http.HandlerFunc(rootHandler)).Methods("GET")
	http.Handle("/", middle)

	err := http.ListenAndServe(address, nil)
	if err != nil {
		println("Can't start HTTP server: " + err.Error())
		os.Exit(1)
	}
}
