# tjg-crypto
RPC service to solve cryptograms

For the moment, I'm using a placeholder helloworld.

## Instructions

To run for the first time, run:
```
cd go-service
go mod tidy
```

To update the proto generated files:  Run `protoc --go_out=proto --go-grpc_out=. ../proto/service.proto --proto_path=../proto` from `go-service`.  (May need to install some packages.)

To test, start up the docker containers, then:
- Create user with `grpcurl -plaintext -d '{"name": "Alice"}' localhost:50051 crud.UserService/CreateUser`
- Get user with `grpcurl -plaintext -d '{"id": 1}' localhost:50051 crud.UserService/GetUser`
