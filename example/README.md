# Example Go client

This is an example Go client to communicate with QEMU instance via QMP.

## Build

```shell
make
```

## Run

1. Start a QEMU Instance: Launch a QEMU instance with QMP enabled. You can use the provided script:
```shell
./start-vm.sh
```
2. Run the Client: Execute the compiled Go client binary to connect to the QEMU instance via QMP:
```shell
./build/example --socket /tmp/example.qmp
```
