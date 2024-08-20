## raftrv

This is a modified version of the etcd [raftexample](https://github.com/etcd-io/etcd/tree/main/contrib/raftexample) to test
using raft for managing a distributed resource version.

### Usage

* install [goreman](https://github.com/mattn/goreman)
* install [k6](https://k6.io/docs/getting-started/installation/)

```sh
make run
```

In another terminal:

```sh
make test
```

### Results 

```
make test
k6 run k6.js

          /\      |‾‾| /‾‾/   /‾‾/   
     /\  /  \     |  |/  /   /  /    
    /  \/    \    |     (   /   ‾‾\  
   /          \   |  |\  \ |  (‾)  | 
  / __________ \  |__| \__\ \_____/ .io

     execution: local
        script: k6.js
        output: -

     scenarios: (100.00%) 1 scenario, 10 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 50000 iterations shared among 10 VUs (maxDuration: 30s, gracefulStop: 30s)


     data_received..................: 4.5 MB 6.9 MB/s
     data_sent......................: 6.5 MB 10 MB/s
     http_req_blocked...............: avg=1.11µs   min=320ns   med=900ns    max=320.89µs p(90)=1.46µs   p(95)=1.75µs  
     http_req_connecting............: avg=36ns     min=0s      med=0s       max=100.16µs p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=87.97µs  min=26.01µs med=65.78µs  max=1.85ms   p(90)=122.83µs p(95)=206.24µs
       { expected_response:true }...: avg=87.97µs  min=26.01µs med=65.78µs  max=1.85ms   p(90)=122.83µs p(95)=206.24µs
     http_req_failed................: 0.00%  ✓ 0            ✗ 50000
     http_req_receiving.............: avg=8.47µs   min=2.33µs  med=7.07µs   max=709.15µs p(90)=11.01µs  p(95)=12.91µs 
     http_req_sending...............: avg=5.55µs   min=1.72µs  med=4.76µs   max=406.24µs p(90)=7µs      p(95)=8.22µs  
     http_req_tls_handshaking.......: avg=0s       min=0s      med=0s       max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=73.94µs  min=19.02µs med=53.36µs  max=1.83ms   p(90)=105.48µs p(95)=171.72µs
     http_reqs......................: 50000  76129.905324/s
     iteration_duration.............: avg=126.18µs min=48.55µs med=100.62µs max=1.92ms   p(90)=173.57µs p(95)=280.09µs
     iterations.....................: 50000  76129.905324/s


running (0m00.7s), 00/10 VUs, 50000 complete and 0 interrupted iterations
contacts ✓ [============] 10 VUs  00.7s/30s  50000/50000 shared iters
```