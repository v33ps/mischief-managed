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
	CommandType    int               `json:"commandType"`
	Function       string            `json:"function"`
	Iterations     int               `json:"iterations"`
	IterationDelay int               `json:"iterationDelay"`
	Params         map[string]string `json:"params"`
	TaskID         int               `json:"taskID"`
	State          int               `json:"state"`
	Block          int               `json:"block"`
}

type Client struct {
	ClientID        int       `json:"clientID"`
	ClientName      string    `json:"clientName"`
	LastCheckinTime int64     `json:"lastcheckintime"`
	TaskQueue       []Command `json:"taskQueue"`
	Interval        float32   `json:"interval"`
}

type FuckYou struct {
	Records          []Client `json:"data"`
	QueryRecordCount int      `json:"queryRecordCount"`
	TotalRecordCount int      `json:"totalRecordCount"`
}

var clientList = []Client{}
var clientIDCounter = 0
var taskIDCounter = 0

func main() {

	router := mux.NewRouter()

	// user facing endpoints
	router.HandleFunc("/", indexPage).Methods("GET")
	router.HandleFunc("/sendPage", sendPage).Methods("GET")
	router.HandleFunc("/send", handleSend).Methods("POST")
	router.HandleFunc("/updateClientList", handleUpdateClientList).Methods("GET")

	// client facing endpionts
	router.HandleFunc("/client/new", clientHandleNew).Methods("POST")
	router.HandleFunc("/client/get_tasks", clientHanldGetTasks).Methods("POST")

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

func handleUpdateClientList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	fuck := FuckYou{}
	fuck.Records = clientList
	fuck.QueryRecordCount = len(clientList)
	fuck.TotalRecordCount = len(clientList)
	b, err := json.Marshal(fuck)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))
	if err := json.NewEncoder(w).Encode(clientList); err != nil {
		panic(err)
	}
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

	cmd := Command{TaskID: taskIDCounter, IterationDelay: 0}
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
	taskIDCounter = taskIDCounter + 1
}

// take the @msg from the webapp and add it to the clients queue
func addToOutgoingQueue(cmd Command, clientID int) error {
	for i, v := range clientList {
		if v.ClientID == clientID {
			// we found our client, now add the msg to their outgoingQueue
			clientList[i].TaskQueue = append(v.TaskQueue, cmd)
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
	client.TaskQueue = make([]Command, 0)
	client.Interval = 0.5
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
	fmt.Println("client is: ", client.ClientID)
	clientIndex := 0
	// search for our client
	for i, v := range clientList {
		if v.ClientID == client.ClientID {
			// we found our client, now add the msg to their outgoingQueue
			client.TaskQueue = v.TaskQueue
			clientIndex = i
			break
		}
	}
	if len(client.TaskQueue) > 0 {
		fmt.Println("client has a task queue: ", client)

		// send the queue back to the client
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		// w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(client); err != nil {
			panic(err)
		}

		// since we've given away the outgoingQueue list, truncate the list
		clientList[clientIndex].TaskQueue = make([]Command, 0)
	} else {
		w.WriteHeader(204)
		w.Write([]byte("HTTP status code returned!"))
	}

}
