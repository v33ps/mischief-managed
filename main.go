package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

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
	CommandType int               `json:"commandType"`
	Function    string            `json:"function"`
	Iterations  int               `json:"iterations"`
	Params      map[string]string `json:"params"`
	TaskID      int               `json:"taskID"`
}

type Client struct {
	ClientID        int       `json:"clientID"`
	ClientName      string    `json:"clientName"`
	LastCheckinTime int64     `json:"lastcheckintime"`
	OutgoingQueue   []Command `json:"outgoingQueue"`
}

var clientList = []Client{}
var clientIDCounter = 0

func main() {

	router := mux.NewRouter()

	// user facing endpoints
	router.HandleFunc("/", indexPage).Methods("GET")
	router.HandleFunc("/sendPage", sendPage).Methods("GET")
	router.HandleFunc("/send", handleSend).Methods("POST")

	// client facing endpionts
	router.HandleFunc("/client/new", clientHandleNew).Methods("POST")
	router.HandleFunc("/client/get_tasks", clientHanldGetTasks).Methods("GET")

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
	clientID, err := strconv.Atoi(r.Form.Get("clientID"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	cmd := Command{}
	if err := json.Unmarshal([]byte(msg), &cmd); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}
	err = addToOutgoingQueue(cmd, clientID)
	if err != nil {
		fmt.Println("oh noooo")
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode("{\"status\":\"ok\"}"); err != nil {
		panic(err)
	}
}

// take the @msg from the webapp and add it to the clients queue
func addToOutgoingQueue(cmd Command, clientID int) error {
	for i, v := range clientList {
		if v.ClientID == clientID {
			// we found our client, now add the msg to their outgoingQueue
			clientList[i].OutgoingQueue = append(v.OutgoingQueue, cmd)
			break
		}
	}
	fmt.Println(clientList)
	return nil
}

// accept a new client and give them their client ID
func clientHandleNew(w http.ResponseWriter, r *http.Request) {
	client := Client{}
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(body, &client); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	clientIDCounter = clientIDCounter + 1

	client.LastCheckinTime = time.Now().Unix()
	client.ClientID = clientIDCounter
	client.OutgoingQueue = make([]Command, 0)
	fmt.Println("client:", client)
	clientList = append(clientList, client)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(client); err != nil {
		panic(err)
	}
}

// search through @clientList to find if our client has anything in the outgoingQueue
func clientHanldGetTasks(w http.ResponseWriter, r *http.Request) {
	// get our client from the incoming JSON
	client := Client{}
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(body, &client); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	clientIndex := 0
	// search for our client
	for i, v := range clientList {
		if v.ClientID == client.ClientID {
			// we found our client, now add the msg to their outgoingQueue
			client.OutgoingQueue = v.OutgoingQueue
			clientIndex = i
			break
		}
	}
	fmt.Println("client: ", client)

	// send the queue back to the client
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	// w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(client); err != nil {
		panic(err)
	}

	// since we've given away the outgoingQueue list, truncate the list
	clientList[clientIndex].OutgoingQueue = make([]Command, 10)
}
