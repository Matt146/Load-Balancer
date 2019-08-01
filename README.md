# Load-Balancer
A minimalistic round robin load balancer written in Go.

# How-To-Start
1. Define all instances of the server in the servers.txt file (make sure that it is in the same directory as loadbalancer.go)

Example of servers.txt file:
```
localhost:81
localhost:82
localhost:83
localhost:84
localhost:85
```

2. Start the instances of your server.
3. Start the load balancer (sudo go run loadbalancer.go) 
   
   Also, it binds to port 80.
