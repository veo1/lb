# Go Load Balancer

This is a simple load balancer that I wrote as a personal project. It supports basic load balancing strategies such as round-robin (RR) and weighted round-robin (WRR), ensuring efficient distribution of incoming HTTP requests among a pool of backend servers. Additionally, it features health checks to monitor the availability of backends and maintains detailed statistics on request handling.
This project is WIP, I plan to implement more load-balancing strategies, support config file and cover it with e2e tests.

## Features

- **Load Balancing Strategies**: Round-robin and weighted round-robin.
- **Health Checks**: Periodic checks to ensure backends are available.
- **Statistics**: Tracks request count, error count, and total latency.
- **Log Rotation**: Uses `lumberjack` for log management.


### Usage

Run the load balancer with the following command:

```
./loadbalancer -servers http://server1.example.com,http://server2.example.com -port 3030 -method rr
```

- `-servers` is a comma-separated list of backend servers.
- `-port` specifies the port on which the load balancer will listen.
- `-method` selects the load balancing method (`rr` for round-robin, `wrr` for weighted round-robin).

### Health Checks and Statistics

- Health checks are performed every 2 minutes to ensure backend servers are reachable.
- Statistics are logged every 10 seconds to `stats.txt`, including request count, errors, and latency.