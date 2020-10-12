clean:
	rm -f rpc/service.pb.go

compile-pb: clean
	protoc --proto_path=rpc --go-grpc_out=rpc --go_out=rpc --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative rpc/service.proto

install: compile-pb
	go install pirosb3/real_feed

build: compile-pb
	go build

test: compile-pb
	go test pirosb3/real_feed/controller
	go test pirosb3/real_feed/feed