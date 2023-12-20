package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	Address   string
	StartPort int
	EndPort   int
}

type Node struct {
	ID    string `json:"id"`
	Group int    `json:"group"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Result struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

func updateResult(theMap map[string][]string) Result {
	result := Result{}

	// keys of theMap
	keys := []string{}
	for k := range theMap {
		keys = append(keys, k)
	}
	// sort keys
	sort.Strings(keys)

	for _, node := range keys {
		links := theMap[node]
		result.Nodes = append(result.Nodes, Node{ID: node, Group: 0})
		for _, link := range links {
			result.Links = append(result.Links, Link{Source: node, Target: link})
		}
	}
	return result
}

func main() {
	fmt.Printf("Hello\n")

	servers := []Server{}

	srvSpec := os.Args[1:]
	// is len(srvSpec) divisible by 3
	if len(srvSpec)%3 != 0 {
		log.Fatalf("need arguments 3 at a time, e.g. " +
			"10.0.0.1 10000 10099 10.0.0.2 10000 10099 10.0.0.3 10000 10099")
	}

	numServers := len(srvSpec) / 3
	for i := 0; i < numServers; i++ {
		start, err := strconv.Atoi(srvSpec[i*3+1])
		if err != nil {
			log.Fatalf("can't interpret start port %s as uint: %s", srvSpec[i+1], err)
		}
		end, err := strconv.Atoi(srvSpec[i*3+2])
		if err != nil {
			log.Fatalf("can't interpret end port %s as uint: %s", srvSpec[i+2], err)
		}
		servers = append(servers, Server{
			Address:   srvSpec[i*3],
			StartPort: start,
			EndPort:   end,
		})
	}

	fmt.Printf("servers: %+v\n", servers)

	theMap := map[string][]string{}
	theResult := Result{}
	// for each server, a list of servers it is connected to
	var theMutex sync.Mutex
	go func() {
		for {
			for _, server := range servers {
				for port := server.StartPort; port <= server.EndPort; port++ {
					addr := fmt.Sprintf("http://%s:%d/", server.Address, port)
					resp, err := http.Get(addr + "/id")
					if err != nil {
						log.Print(err)
						continue
					}
					newID := ""
					err = json.NewDecoder(resp.Body).Decode(&newID)
					if err != nil {
						log.Print(err)
						continue
					}
					resp.Body.Close()

					resp, err = http.Get(addr + "/peers")
					if err != nil {
						log.Print(err)
						continue
					}
					newList := map[string][]string{}
					err = json.NewDecoder(resp.Body).Decode(&newList)
					if err != nil {
						log.Print(err)
						continue
					}
					resp.Body.Close()

					func() {
						theMutex.Lock()
						defer theMutex.Unlock()
						theMap[newID] = newList["bacalhau-job-event"]
						sort.Strings(theMap[newID])

						theResult = updateResult(theMap)
					}()
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()

	// serve local files on web server

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.Handle("/api/map", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		theMutex.Lock()
		defer theMutex.Unlock()
		err := json.NewEncoder(w).Encode(theResult)
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))

	log.Print("Listening on :31337...")
	err := http.ListenAndServe(":31337", nil)
	if err != nil {
		log.Fatal(err)
	}
}
