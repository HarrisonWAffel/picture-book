FROM golang:1.19-alpine

WORKDIR /

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./

RUN apk add --no-cache --upgrade bash

RUN go build -o picture-book

EXPOSE 8001


CMD [ "./picture-book", "sync" ]