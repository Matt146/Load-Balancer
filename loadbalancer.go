package main

import (
    "fmt"
    "log"
    "net/http"
    "sync"
    "strconv"
    "time"
    "io/ioutil"
    "strings"
    "os"
)

var wg sync.WaitGroup

const (
    MAX_SERVERS = 128
    LB_SECRET	= "744AE219425641E546593FA04B03D47D0468718CD0C093AA2AA61EF59739DC75"
)

type Balancer struct {
    ServerIPs []string      /*A list of the server IP's*/
    ServerErrors map[string]uint32
    CurrentServer int    /*Which server IP we will send the next request to*/
    RequestCount int64     /*The number of requests processed*/
}

/*RemoveFromSlice: Removes an element from a slice based off of its index*/
func RemoveFromSlice(slice []string, s int) []string {
    return append(slice[:s], slice[s+1:]...)
}

/*MakeBalancer: Initializes the load balancer*/
func MakeBalancer() (b *Balancer) {
    balancer := &Balancer{ServerIPs: make([]string, 0, 128), ServerErrors: make(map[string]uint32), CurrentServer: 0, RequestCount: 0}
    return balancer
}

func (b *Balancer) ReadServerList(fname string) {
    f, err := os.Open(fname)
    defer f.Close();
    if err != nil {
        log.Println("type=error function=Balancer.ReadServerList(); value=fopen-error from=nil id=nil")
    }
    fdata, err := ioutil.ReadAll(f)
    if err != nil {
        log.Println("type=error function=Balancer.ReadServerList(); value=fread-error from=nil id=nil")
    }
    fdataStr := string(fdata)
    fdataStrSplit := strings.Split(fdataStr, "\n")
    for _, v := range fdataStrSplit {
        b.ServerIPs = append(b.ServerIPs, v)
    }
}

/*GeneralHandleFunc: All HTTP requests are routed through here.*/
func (b *Balancer) GeneralHandleFunc(w http.ResponseWriter, r *http.Request) {
    wg.Add(1)
    if b.ServerIPs[0] == "" {
        panic("[Error]: All Servers are dead!")
    }
    fmt.Printf("\n[Servers] %v\n", b.ServerIPs)
    fmt.Printf("\t%s\n", b.ServerIPs[b.CurrentServer])
    log.Println("type=incoming function=Balancer.GeneralHandleFunc(); value=newrequest from=" + r.RemoteAddr + " id=" + strconv.FormatInt(b.RequestCount, 16))
    request, err := http.NewRequest(r.Method, "http://" + b.ServerIPs[b.CurrentServer] + r.URL.Path, r.Body)
    if err != nil {
        log.Println("type=error function=Balancer.GeneralHandleFunc(); value=badrequest from=" + r.RemoteAddr + " id=" + strconv.FormatInt(b.RequestCount, 16))
        w.WriteHeader(http.StatusBadRequest)
        b.ServerErrors[b.ServerIPs[b.CurrentServer]] += 1
        if b.ServerErrors[b.ServerIPs[b.CurrentServer]] >= 3 {
            log.Println("type=server-removed function=Balancer.GeneralHandleFunc(); value=" + b.ServerIPs[b.CurrentServer] + "from=" + r.RemoteAddr + " id=" + strconv.FormatInt(b.RequestCount, 16))
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
        log.Println("type=error function=Balancer.GeneralHandleFunc(); value=badresponse from=" + r.RemoteAddr + " id=" + strconv.FormatInt(b.RequestCount, 16))
        fmt.Printf("\t" + err.Error())
        w.WriteHeader(http.StatusInternalServerError)
        b.ServerErrors[b.ServerIPs[b.CurrentServer]] += 1
        if b.ServerErrors[b.ServerIPs[b.CurrentServer]] >= 3 {
            log.Println("type=server-removed function=Balancer.GeneralHandleFunc(); value=" + b.ServerIPs[b.CurrentServer] + "from=" + r.RemoteAddr + " id=" + strconv.FormatInt(b.RequestCount, 16))
            RemoveFromSlice(b.ServerIPs, b.CurrentServer)
        }
        return
    }
    defer response.Body.Close()
    responseBytes, err := ioutil.ReadAll(response.Body)
    if err != nil {
        log.Println("type=error function=Balancer.GeneralHandleFunc(); value=badresponse from=" + r.RemoteAddr + " id=" + strconv.FormatInt(b.RequestCount, 16))
        fmt.Printf("\t" + err.Error())
        w.WriteHeader(http.StatusInternalServerError)
        b.ServerErrors[b.ServerIPs[b.CurrentServer]] += 1
        if b.ServerErrors[b.ServerIPs[b.CurrentServer]] >= 3 {
            log.Println("type=server-removed function=Balancer.GeneralHandleFunc(); value=" + b.ServerIPs[b.CurrentServer] + "from=" + r.RemoteAddr + " id=" + strconv.FormatInt(b.RequestCount, 16))
            RemoveFromSlice(b.ServerIPs, b.CurrentServer)
        }
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write(responseBytes)
    log.Println("type=success function=Balancer.GeneralHandleFunc(); value=requestforwarded from=" + r.RemoteAddr + " id=" + strconv.FormatInt(b.RequestCount, 16))
    b.CurrentServer += 1
    if b.CurrentServer >= len(b.ServerIPs)-1 || b.CurrentServer >= cap(b.ServerIPs){
        b.CurrentServer = 0
    }
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
    log.Println("type=startup function=main(); value=startingserver! from=nil id=STARTUP")
    b := MakeBalancer()
    b.ReadServerList("servers.txt")
    http.HandleFunc("/", b.GeneralHandleFunc)
    http.ListenAndServe(":80", nil)
}
