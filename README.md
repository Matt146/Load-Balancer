# Load-Balancer
A minimalistic round robin load balancer written in Go.

# How-To-Start
1. Define all instances of the server in the servers.yml file (make sure that it is in the same directory as loadbalancer.go)

Example of servers.yml file:
```
servers:
   - "localhost:81"
   - "localhost:82"
   - "localhost:83"
   - "localhost:84"
   - "localhost:85"
```

2. Start the instances of your server.
3. Build the load balancer (go build loadbalancer.go)
   
4. Run the build
   
   Also, keep in mind that it binds to port 80
