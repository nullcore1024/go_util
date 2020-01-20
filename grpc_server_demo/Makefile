gen:
	protoc --go_out=. hello/hello.proto
all:
	go build server.go
	go build client.go
grpc:
	protoc    --go_out=plugins=grpc:.  hello/hello.proto
clean:
	rm -rf server client
	make clean -C hello
