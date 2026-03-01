build:
	./build.sh

test: build
	./src/bin/gomon -w -v -s
