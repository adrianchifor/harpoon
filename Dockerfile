FROM golang:1.14-alpine as builder

RUN apk add --no-cache curl git gcc musl-dev docker

RUN curl -fsSL "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v2.0.1/docker-credential-gcr_linux_amd64-2.0.1.tar.gz" \
  | tar xz --to-stdout ./docker-credential-gcr > /bin/docker-credential-gcr && chmod +x /bin/docker-credential-gcr

WORKDIR /go/src/harpoon
COPY *.go go.mod /go/src/harpoon/

RUN go mod download

RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/harpoon

# Runner
FROM alpine

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY --from=builder /go/bin/harpoon /
COPY --from=builder /bin/docker-credential-gcr /bin/
COPY --from=builder /usr/bin/docker /bin/

ENTRYPOINT ["/harpoon"]
