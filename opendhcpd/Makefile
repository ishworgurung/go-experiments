GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

bin:
	GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags '-s -w' -o opendhcpd

clean:
	@rm -f opendhcpd

