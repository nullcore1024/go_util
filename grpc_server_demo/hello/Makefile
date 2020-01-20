gen:
	protoc --go_out=. hello.proto
all:
	echo "abc"
grpc:
	protoc    --go_out=plugins=grpc:.  hello.proto
clean:
	rm -rf *.go
