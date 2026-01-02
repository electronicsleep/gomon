build:
	go build -o gomon gomon.go

linux:
	./build.sh

test: build
	./gomon -w -v -s
