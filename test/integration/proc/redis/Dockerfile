FROM golang:1.13-alpine

RUN apk add redis>5.0.5 bash

ENV CGO_ENABLED=0
ENV SAM_PATH=/samaritan
COPY . $SAM_PATH/
WORKDIR $SAM_PATH
