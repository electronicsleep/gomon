build:
	go build -o gomon gomon.go
	./gomon -w -v -s

linux:
	./build.sh
