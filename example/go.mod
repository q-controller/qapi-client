module qga-example

go 1.25.5

replace github.com/q-controller/qapi-client v0.0.0 => ../

require (
	github.com/google/uuid v1.6.0
	github.com/q-controller/qapi-client v0.0.0
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.34.0 // indirect
)
