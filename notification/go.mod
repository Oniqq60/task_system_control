module github.com/Oniqq60/task_system_control/notification

go 1.23.4

require (
	github.com/Oniqq60/task_system_control/gen/proto/auth v0.0.0
	github.com/joho/godotenv v1.5.1
	github.com/segmentio/kafka-go v0.4.49
	google.golang.org/grpc v1.64.0
)

require (
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/Oniqq60/task_system_control/gen/proto/auth => ../gen/proto/auth
