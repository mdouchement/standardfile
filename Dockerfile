# build stage
FROM golang:1.12-alpine as build-env
MAINTAINER mdouchement

RUN apk upgrade
RUN apk add --update --no-cache git curl

RUN curl -sL https://raw.githubusercontent.com/go-task/task/master/install-task.sh | sh

RUN mkdir -p /go/src/github.com/mdouchement/standardfile
WORKDIR /go/src/github.com/mdouchement/standardfile

ENV CGO_ENABLED 0
ENV GO111MODULE on
ENV GOPROXY https://proxy.golang.org

COPY . /go/src/github.com/mdouchement/standardfile
# Dependencies
RUN go mod download

RUN task build-server

# final stage
FROM alpine
MAINTAINER mdouchement

ENV DATABASE_PATH /data/database

RUN mkdir -p ${DATABASE_PATH}

COPY --from=build-env /go/src/github.com/mdouchement/standardfile/dist/standardfile /usr/local/bin/

EXPOSE 5000
CMD ["standardfile", "server", "-p", "5000"]
# CMD ["standardfile", "server", "-p", "5000", "--noreg"]
