build:
	./build.sh

test: build
	./bin/gomon -w -v -s
