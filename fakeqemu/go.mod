module github.com/krjakbrjak/fakeqemu

go 1.27.3

replace github.com/q-controller/qapi-client v0.0.0 => ../

require (
	github.com/google/uuid v1.6.0
	github.com/q-controller/qapi-client v0.0.0
)

require golang.org/x/sys v0.48.0 // indirect
