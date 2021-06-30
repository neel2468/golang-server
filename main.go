package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type counters struct {
	sync.Mutex
	view  int
	click int
}

var (
	c       = counters{}
	content = []string{"sports", "entertainment", "business", "education"}

	// A map describing a structure for storing counter values.
	// In this structure the key is string and the value is again a map consisting of key  value pair
	// in which "views" and "clicks" can be keys and their values are integer
	storeCounters = make(map[string]map[string]int)
	//Here I have used go lang rate which implements leaf bucket
	//algorithm for limiting the http requests and here have created a
	// bucket size of 4 tokens per second with burst size of 6
	limiter = rate.NewLimiter(4, 6)
)


func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to EQ Works 😎")
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	data := content[rand.Intn(len(content))]

	c.Lock()
	c.view++
	c.Unlock()

	err := processRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	// simulate random click call
	if rand.Intn(100) < 50 {
		processClick(data)
	}

}

func processRequest(r *http.Request) error {
	time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
	return nil
}

func processClick(data string) error {
	c.Lock()
	c.click++
	c.Unlock()
	return nil
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAllowed() {
		w.WriteHeader(429)
		w.Write([]byte("Too many requests"))
		return
	} else {
		// Reading the data from the mock store i.e. data.json
		jsonData, err := ioutil.ReadFile("data.json")
		// if any error occurs return appropriate message else data
		if err != nil {
			fmt.Println(err.Error())
			w.Write([]byte("No data found"))

		} else {
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(200)
			w.Write(jsonData)
		}
	}

}

func isAllowed() bool {
	if limiter.Allow() == false {
		return false
	} else {
		return true
	}

}

func uploadCounters() error {
	// Selecting random value from the string of data
	data := content[rand.Intn(len(content))]
	// Getting current time
	currentDateTime := time.Now()
	// Appending the string and current time in yyyy-mm-dd hh:mm:ss format to have
	// a unique key
	key := data + ":" + currentDateTime.Format("2006-01-02 15:04:10")
	// Check if key in map alreay exists or not
	// if not then generate a map of having views and clicks with default value of zero
	if _, ok := storeCounters[key]; !ok {
		storeCounters[key] = map[string]int{
			"views":  c.view + 0,
			"clicks": c.click + 0,
		}
	}
	//Here a assumption is made that upload counter can also increment views and clicks besides view handler
	// increment views and clicks
	c.Lock()
	c.view++
	c.click++
	c.Unlock()
	// convert storeCounter map to json to be able to write in json file
	// which in this case is the mock store for storing counter's values
	jsonData, err1 := json.Marshal(storeCounters)
	if err1 != nil {
		fmt.Println(err1.Error())
	}
	// Create a file for writing if not exists or write into existing if it already exists
	jsonFile, err2 := os.OpenFile("data.json", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err2 != nil {
		fmt.Println(err2)
	}
	// Write the json content to the file
	_, err3 := jsonFile.Write(jsonData)
	if err3 != nil {
		fmt.Println(err3.Error())
	}
	// close file pointer
	defer jsonFile.Close()

	return nil
}

// a function which will call a specified function every n seconds
func call_upload_counters(d time.Duration, f func() error) {
	for range time.Tick(d) {
		f()
	}
}

func main() {
	// A go routine which will call upload counter func every 5 seconds
	go call_upload_counters(5, uploadCounters)

	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/stats/", statsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
