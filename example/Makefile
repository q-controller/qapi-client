.PHONY: clean example generate all

BUILD_DIR=build

all: example

clean:
	rm -fr ${BUILD_DIR}
	find . -name "*.pb.go" -exec rm -f {} \;

example:
	mkdir -p ${BUILD_DIR}
	../generate.sh --schema ../qemu/qapi/qapi-schema.json --out-dir generated --package qapi
	go build -o ${BUILD_DIR}/example main.go
