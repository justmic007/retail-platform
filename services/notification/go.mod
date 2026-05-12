module retail-platform/notification

go 1.25.0

require (
	github.com/caarlos0/env/v10 v10.0.0
	retail-platform/pkg v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/redis/go-redis/v9 v9.19.0 // indirect
	github.com/rs/zerolog v1.32.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace retail-platform/pkg => ../../pkg
