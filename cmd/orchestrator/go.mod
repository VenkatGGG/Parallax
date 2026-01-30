module github.com/microcloud/orchestrator

go 1.23

require (
	connectrpc.com/connect v1.18.1
	github.com/microcloud/bus v0.0.0
	github.com/microcloud/gen/go v0.0.0
	github.com/microcloud/logger v0.0.0
	github.com/microcloud/storage v0.0.0
	golang.org/x/net v0.34.0
	golang.org/x/sync v0.10.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.2 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
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
	github.com/microcloud/storage => ../../pkg/storage
)
