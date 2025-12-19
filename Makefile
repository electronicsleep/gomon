build:
	go build -o bin/gomon gomon.go
	./gomon -w -v -s

linux:
	./build.sh


