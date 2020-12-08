all:
	CGO_ENABLED=0 go build . -a -ldflags '-extldflags "-static"' 
app:
	go build -a -ldflags '-extldflags "-static"' .
build:
	CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' .
