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