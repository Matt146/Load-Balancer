package main

import (
    "fmt"
    "log"
    "net/http"
    "sync"
    "strconv"
    "time"
    "io/ioutil"
    "os"

    "github.com/go-yaml/yaml"
)

var wg sync.WaitGroup

const (
    MAX_SERVERS = 10240
)

type YAMLServerList struct {
    ServerList []string `yaml:"servers"`
}

type Balancer struct {
    ServerIPs []string      /*A list of the server IP's*/
    ServerErrors map[string]uint32
    CurrentServer int    /*Which server IP we will send the next request to*/
    RequestCount int64     /*The number of requests processed*/
}

/*LogEvent: Used to log ALL events and keep structured logging present*/
func LogEvent(typeOf string, function string, value string, id string) {
    log.Printf("type=%s, function=%s, value=%v, id=%s\n", typeOf, function, value, id)
}

/*RemoveFromSlice: Removes an element from a slice based off of its index*/
func RemoveFromSlice(slice []string, i int) []string{
    if i >= len(slice){
        return slice
    }
    slice[i], slice[len(slice)-1] = slice[len(slice)-1], slice[i]
    return slice[:len(slice)-1]
}

/*MakeBalancer: Initializes the load balancer*/
func MakeBalancer() (b *Balancer) {
    balancer := &Balancer{ServerIPs: make([]string, 0, MAX_SERVERS), ServerErrors: make(map[string]uint32), CurrentServer: 0, RequestCount: 0}
    return balancer
}

func (b *Balancer) ReadServerList(fname string) {
    f, err := os.Open(fname)
    defer f.Close();
    if err != nil {
        LogEvent("error", "Balancer.ReadServerList", "Unable to open server list file: " + err.Error(), "nil")
    }
    fdata, err := ioutil.ReadAll(f)
    if err != nil {
        LogEvent("error", "Balancer.ReadServerList", "Unable to read server list: " + err.Error(), "nil")
    }
    serverList := YAMLServerList{make([]string, 0, MAX_SERVERS)}
    err = yaml.Unmarshal(fdata, &serverList)
    if err != nil {
        LogEvent("error", "Balancer.ReadServerList", "Unable to parse YAML in server list: " + err.Error(), "nil")
    }
    for _, v := range serverList.ServerList{
        b.ServerIPs = append(b.ServerIPs, v)
    }
    fmt.Printf("\n\n%v\n\n", b.ServerIPs)
    fmt.Printf("\n\n%v\n\n", serverList.ServerList)
    LogEvent("success", "Balancer.ReadServerList", "Read server list!", "nil")
}

/*GeneralHandleFunc: All HTTP requests are routed through here.*/
func (b *Balancer) GeneralHandleFunc(w http.ResponseWriter, r *http.Request) {
    wg.Add(1)
    if b.ServerIPs[0] == "" {
        LogEvent("crash", "Balancer.GeneralHandleFunc", "All servers down!", "nil")
    }
    fmt.Printf("\n[Servers] %v\n", b.ServerIPs)
    fmt.Printf("\t%s\n", b.ServerIPs[b.CurrentServer])
    LogEvent("incoming", "Balancer.GeneralHandleFunc", r.Method + " " + r.URL.Path + " (" + r.RemoteAddr + ")", strconv.FormatInt(b.RequestCount, 16))
    request, err := http.NewRequest(r.Method, "http://" + b.ServerIPs[b.CurrentServer] + r.URL.Path, r.Body)
    if err != nil {
        LogEvent("error", "Balancer.GeneralHandleFunc", "Unable to initialize request to application server: " + err.Error(), strconv.FormatInt(b.RequestCount, 16))
        w.WriteHeader(http.StatusBadRequest)
        b.ServerErrors[b.ServerIPs[b.CurrentServer]] += 1
        if b.ServerErrors[b.ServerIPs[b.CurrentServer]] >= 3 {
            LogEvent("server-removed", "Balancer.GeneralHandleFunc", b.ServerIPs[b.CurrentServer], strconv.FormatInt(b.RequestCount, 16))
            RemoveFromSlice(b.ServerIPs, b.CurrentServer)
        }
        return
    }
    for key, value := range r.Header {
        for _, v := range value {
            request.Header.Add(key, v)
        }
    }

    client := http.Client{Timeout: time.Second * 3}
    response, err := client.Do(request)
    if err != nil {
        LogEvent("error", "Balancer.GeneralHandleFunc", "Unable to send request to application server: " + err.Error(), strconv.FormatInt(b.RequestCount, 16))
        fmt.Printf("\t" + err.Error())
        w.WriteHeader(http.StatusInternalServerError)
        b.ServerErrors[b.ServerIPs[b.CurrentServer]] += 1
        if b.ServerErrors[b.ServerIPs[b.CurrentServer]] >= 3 {
            LogEvent("server-removed", "Balancer.GeneralHandleFunc", b.ServerIPs[b.CurrentServer], strconv.FormatInt(b.RequestCount, 16))
            RemoveFromSlice(b.ServerIPs, b.CurrentServer)
        }
        return
    }
    defer response.Body.Close()
    responseBytes, err := ioutil.ReadAll(response.Body)
    if err != nil {
        LogEvent("error", "Balancer.GeneralHandleFunc", "Unable to read response from application server: " + err.Error(), strconv.FormatInt(b.RequestCount, 16))
        fmt.Printf("\t" + err.Error())
        w.WriteHeader(http.StatusInternalServerError)
        b.ServerErrors[b.ServerIPs[b.CurrentServer]] += 1
        if b.ServerErrors[b.ServerIPs[b.CurrentServer]] >= 3 {
            LogEvent("server-removed", "Balancer.GeneralHandleFunc", b.ServerIPs[b.CurrentServer], strconv.FormatInt(b.RequestCount, 16))
            RemoveFromSlice(b.ServerIPs, b.CurrentServer)
        }
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write(responseBytes)
    LogEvent("success", "Balancer.GeneralHandleFunc", "Succeeded in sending request: " + r.Method + " " + r.URL.Path + " (" + r.RemoteAddr + ")", strconv.FormatInt(b.RequestCount, 16))
    b.CurrentServer += 1
    if b.CurrentServer == len(b.ServerIPs) {
        b.CurrentServer = 0
    }
    b.RequestCount += 1
    wg.Done()
}

func WelcomeMessage() {
    fmt.Print(`
██╗      ██████╗  █████╗ ██████╗     ██████╗  █████╗ ██╗      █████╗ ███╗   ██╗ ██████╗███████╗██████╗
██║     ██╔═══██╗██╔══██╗██╔══██╗    ██╔══██╗██╔══██╗██║     ██╔══██╗████╗  ██║██╔════╝██╔════╝██╔══██╗
██║     ██║   ██║███████║██║  ██║    ██████╔╝███████║██║     ███████║██╔██╗ ██║██║     █████╗  ██████╔╝
██║     ██║   ██║██╔══██║██║  ██║    ██╔══██╗██╔══██║██║     ██╔══██║██║╚██╗██║██║     ██╔══╝  ██╔══██╗
███████╗╚██████╔╝██║  ██║██████╔╝    ██████╔╝██║  ██║███████╗██║  ██║██║ ╚████║╚██████╗███████╗██║  ██║
╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═════╝     ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝╚══════╝╚═╝  ╚═╝
    `)
    fmt.Println("FAQ:")
    fmt.Println("\t1. How does it work?")
    fmt.Println("\t-Well, it is essentially a reverse proxy that takes the servers listed in servers.txt and updates a list of servers to forward requests to from the client. The response then gets forwarded to the client from the load balancer. This all happens using the round-robin algorithm. Additionally, if a server errors 3 times it is considered down and is removed from the cached list of servers")
    fmt.Println("\t2. How do I set it up?")
    fmt.Println("\t-Very simple actually. First list out all the instances of your server that you want to route to in the servers.txt file. All instances of the server should not include the scheme but just the IP and the port. Additionally, they should be separated by a newline character, '\\n'.")
    fmt.Println("\t3. What if I want to make a change to the server?")
    fmt.Println("\t-That's fine. Just kill the server and the load balancer, make your changes, and start everything over again.")
}

func main() {
    WelcomeMessage()
    LogEvent("startup", "main", "Starting server!", "nil")
    b := MakeBalancer()
    b.ReadServerList("servers.yml")
    http.HandleFunc("/", b.GeneralHandleFunc)
    http.ListenAndServe(":80", nil)
}
