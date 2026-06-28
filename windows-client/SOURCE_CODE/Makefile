.PHONY: build build-gui all
all: build build-gui

build:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ../dist/nossh.exe ./cmd/nossh

build-gui:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
		go build -ldflags="-H windowsgui" -o ../dist/quelo-connect.exe ./cmd/quelo-connect-gui-win
