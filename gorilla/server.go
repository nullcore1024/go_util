package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	hostname string
	port     int
)

/* register command line options */
func init() {
	flag.StringVar(&hostname, "hostname", "0.0.0.0", "The hostname or IP on which the REST server will listen")
	flag.IntVar(&port, "port", 8080, "The port on which the REST server will listen")
}

func MyGetHandler(w http.ResponseWriter, r *http.Request) {
	// parse query parameter
	vals := r.URL.Query()
	param, _ := vals["servicename"] // get query parameters

	// composite response body
	var res = map[string]string{"result": "succ", "name": param[0]}
	response, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func MyGetHandler2(w http.ResponseWriter, r *http.Request) {
	var res map[string]string = make(map[string]string)
	var status = http.StatusOK

	vals := r.URL.Query()
	param, ok := vals["name"]
	if !ok {
		res["result"] = "fail"
		res["error"] = "required parameter name is missing"
		status = http.StatusBadRequest
	} else {
		res["result"] = "succ"
		res["name"] = param[0]
		status = http.StatusOK
	}

	response, _ := json.Marshal(res)
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func MyPostHandler(w http.ResponseWriter, r *http.Request) {
	// parse path variable
	vars := mux.Vars(r)
	servicename := vars["servicename"]

	// parse JSON body
	var req map[string]interface{}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &req)
	servicetype := req["servicetype"].(string)

	// composite response body
	var res = map[string]string{"result": "succ", "name": servicename, "type": servicetype}
	response, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)

}

func main() {
	flag.Parse()
	var address = fmt.Sprintf("%s:%d", hostname, port)
	log.Println("REST service listening on", address)

	// register router
	router := mux.NewRouter().StrictSlash(true)
	router.
		HandleFunc("/api/service/get", MyGetHandler).
		Methods("GET")

	router.
		HandleFunc("/api/service/get2", MyGetHandler2).
		Methods("GET")

	router.
		HandleFunc("/api/service/{servicename}/post", MyPostHandler).
		Methods("POST")

	// start server listening
	err := http.ListenAndServe(address, router)
	if err != nil {
		log.Fatalln("ListenAndServe err:", err)
	}

	log.Println("Server end")
}
