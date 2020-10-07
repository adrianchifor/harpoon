FROM golang:1.14-alpine as builder

RUN apk add --no-cache curl git gcc musl-dev docker

RUN curl -fsSL "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v2.0.1/docker-credential-gcr_linux_amd64-2.0.1.tar.gz" \
  | tar xz --to-stdout > /bin/docker-credential-gcr && chmod +x /bin/docker-credential-gcr

RUN curl -fsSL "https://github.com/kubernetes-sigs/cri-tools/releases/download/v1.19.0/crictl-v1.19.0-linux-amd64.tar.gz" \
  | tar xz --to-stdout > /bin/crictl && chmod +x /bin/crictl

WORKDIR /go/src/harpoon
COPY *.go go.mod /go/src/harpoon/

RUN go mod download

RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/harpoon

# Runner
FROM alpine

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY --from=builder /go/bin/harpoon /
COPY --from=builder /bin/docker-credential-gcr /bin/
COPY --from=builder /bin/crictl /bin/
COPY --from=builder /usr/bin/docker /bin/

ENTRYPOINT ["/harpoon"]
