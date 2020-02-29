package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// this is a global counter that gets incremented each time a new
// record is added to the database
var recordID int
var db *sql.DB
var m sync.Mutex
var templates = template.Must(template.ParseGlob("public/templates/*"))

//Page is used for templating html pages
type Page struct {
	Title  string
	Result string
}

type Command struct {
	taskID      int
	CommandType int
	Function    string
	Iterations  int
	Params      map[string]string
}

type Client struct {
	clientID        int
	outgoingQueue   []Command
	lastCheckinTime int64
}

var clientList = []Client{}

func main() {

	router := mux.NewRouter()

	// user facing endpoints
	router.HandleFunc("/", indexPage).Methods("GET")
	router.HandleFunc("/sendPage", sendPage).Methods("GET")
	router.HandleFunc("/send", handleSend).Methods("POST")

	// client facing endpionts
	router.HandleFunc("/client/new", clientHandleNew).Methods("GET")

	http.Handle("/", router)
	http.ListenAndServe(":7777", nil)
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	display(w, "main", Page{Title: "Home", Result: "test"})
}

func sendPage(w http.ResponseWriter, r *http.Request) {
	display(w, "send", Page{Title: "Send", Result: "test"})
}

func display(w http.ResponseWriter, tmpl string, data interface{}) {
	templates.ExecuteTemplate(w, tmpl, data)
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	msg := r.Form.Get("data")
	fmt.Println(msg)
	data := &Command{}
	err := json.Unmarshal([]byte(msg), data)
	if err != nil {
		fmt.Println("we have an error oh nooo", err)
	}
	fmt.Println(data)
}

// handle a new incoming client
func clientHandleNew(w http.ResponseWriter, r *http.Request) {
	fmt.Println("oh my god hi")
	var newClient Client
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "ERROR parsing JSON data dummy")
	}

	json.Unmarshal(reqBody, &newClient)
	clientList = append(clientList, newClient)
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(newClient)
	fmt.Println(clientList)
	resp := []byte("omg hi")
	w.Write(resp)
}
