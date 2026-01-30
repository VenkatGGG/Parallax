module github.com/microcloud/sim-engine

go 1.23

require (
	connectrpc.com/connect v1.18.1
	github.com/microcloud/bus v0.0.0
	github.com/microcloud/gen/go v0.0.0
	github.com/microcloud/logger v0.0.0
	golang.org/x/net v0.34.0
	golang.org/x/sync v0.10.0
)

require (
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/nats-io/nats.go v1.39.1 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace (
	github.com/microcloud/bus => ../../pkg/bus
	github.com/microcloud/gen/go => ../../gen/go
	github.com/microcloud/logger => ../../pkg/logger
)
