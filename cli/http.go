package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/carbocation/interpose"
	"github.com/carbocation/interpose/middleware"
	"github.com/dogestry/dogestry/config"
	"github.com/gorilla/mux"
)

func errJson(msg string) []byte {
	problem := struct {
		Err string `json:"error"`
	}{
		Err: msg,
	}

	// This is how we generate errors. If an error happens here, well...
	bytes, _ := json.Marshal(problem)
	return bytes
}

func pullHandler(response http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	response.Header().Set("Content-Type", "application/json")

	cfg, err := config.NewServerConfig(req.Header.Get("X-Registry-Auth"))
	if err != nil {
		response.Write(errJson(err.Error()))
		return
	}

	fmt.Println(cfg)

	dogestryCli, err := NewDogestryCli(cfg, make([]string, 0))
	if err != nil {
		response.Write(errJson(err.Error()))
		return
	}

	image := req.URL.Query().Get("fromImage")

	response.Write([]byte(fmt.Sprintf(`{"status": "pulling %s from S3"}`, image)))

	if err := dogestryCli.CmdPull(cfg.AWS.S3URL.String(), image); err != nil {
		fmt.Printf("Ran into errors: %v\n", err)
		response.Write([]byte(`{"status": "error"}`))
		return
	}

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

func ServeHttp(address string) {
	router := mux.NewRouter()
	middle := interpose.New()
	middle.Use(middleware.GorillaLog())
	middle.UseHandler(router)

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
