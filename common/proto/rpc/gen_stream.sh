#!/bin/sh

cd $(dirname $0)

protoc --go_out=. --go-grpc_out=. *.proto