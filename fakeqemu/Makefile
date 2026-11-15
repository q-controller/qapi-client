.PHONY: clean fakeqemu all

BUILD_DIR=build

all: fakeqemu

clean:
	rm -fr ${BUILD_DIR}
	find . -name "*.pb.go" -exec rm -f {} \;

fakeqemu:
	../generate.sh --schema ../qemu/qapi/qapi-schema.json --out-dir generated --package qapi
	go build -o ${BUILD_DIR}/fakeqemu main.go