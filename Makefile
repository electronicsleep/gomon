build:
	./build.sh

test: build
	./gomon -w -v -s
