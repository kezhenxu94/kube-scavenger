# Kubernetes Scavenger

This project helps you to remove namespaces / services / deployments / pods by given label selector after specified delay.

# Usage

1. Start it

```shell
kubectl create namespace kube-scavenger
kubectl -n kube-scavenger apply -f https://raw.githubusercontent.com/istio/istio/release-1.8/samples/bookinfo/platform/kube/bookinfo.yaml
kubectl -n kube-scavenger apply -f https://raw.githubusercontent.com/kezhenxu94/kube-scavenger/main/example/example.yaml
```

1. Connect via TCP

```shell
kubectl -n kube-scavenger exec -it $(kubectl -n kube-scavenger get pods -l app=kube-scavenger -o name) -- nc localhost 8080
```

1. Send some label selectors

```shell
app=productpage
```

1. Close the connection

1. See namespaces / services / deployments / pods deleted after 10s:

```shell
kubectl -n kube-scavenger logs $(kubectl -n kube-scavenger get pods -l app=kube-scavenger -o name) -f
```

```text
2021/02/02 15:22:16 Pinging API Server...
2021/02/02 15:22:16 Connect to API server successfully!
2021/02/02 15:22:16 Starting on port 8080...
2021/02/02 15:22:16 Started!
2021/02/02 15:22:25 New client connected: [::1]:41427
2021/02/02 15:22:25 Received the first connection
2021/02/02 15:22:33 Adding app=productpage
2021/02/02 15:22:34 EOF
2021/02/02 15:22:34 Client disconnected: [::1]:41427
2021/02/02 15:22:44 Timed out waiting for re-connection
2021/02/02 15:22:44 Deleting app=productpage
2021/02/02 15:22:44 services "productpage-v1" not found
2021/02/02 15:22:46 Removed 1 pod(s), 0 deployment(s), 1 service(s) 0 namespace(s)
```
