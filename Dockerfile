# build stage
FROM golang:alpine as build-env
MAINTAINER mdouchement

RUN apk upgrade
RUN apk add --update --no-cache git curl go-task

RUN mkdir -p /go/src/github.com/mdouchement/standardfile
WORKDIR /go/src/github.com/mdouchement/standardfile

ENV CGO_ENABLED 0
ENV GO111MODULE on
ENV GOPROXY https://proxy.golang.org

COPY . /go/src/github.com/mdouchement/standardfile
# Dependencies
RUN go mod download

RUN go-task build-server

# final stage
FROM alpine
MAINTAINER mdouchement

ENV DATABASE_PATH /data/database

RUN mkdir -p ${DATABASE_PATH}

COPY --from=build-env /go/src/github.com/mdouchement/standardfile/dist/standardfile /usr/local/bin/

EXPOSE 5000
CMD ["standardfile", "server", "-c", "/etc/standardfile/standardfile.yml"]
