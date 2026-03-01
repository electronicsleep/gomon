build:
	./src/scripts/build.sh

test: build
	./src/bin/gomon -w -v -s
