FROM golang:1.15 as workspace

WORKDIR /go/src/github.com/kezhenxu94/kube-scavenger

COPY go.mod go.sum ./

COPY . ./

RUN make build

FROM alpine:3

COPY --from=workspace /go/src/github.com/kezhenxu94/kube-scavenger/bin/kube-scavenger /app

CMD ["/app"]
