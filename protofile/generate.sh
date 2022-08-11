#!/bin/sh

 goctl rpc protoc payment.proto --go_out=../pb --go-grpc_out=../pb --zrpc_out=..