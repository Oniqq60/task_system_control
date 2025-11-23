module github.com/Oniqq60/task_system_control/api_gateway

go 1.23.4

require (
	github.com/Oniqq60/task_system_control/gen/proto/auth v0.0.0
	github.com/Oniqq60/task_system_control/gen/proto/document v0.0.0
	github.com/Oniqq60/task_system_control/gen/proto/task v0.0.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/joho/godotenv v1.5.1
	google.golang.org/grpc v1.64.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)

replace github.com/Oniqq60/task_system_control/gen/proto/auth => ../gen/proto/auth

replace github.com/Oniqq60/task_system_control/gen/proto/document => ../gen/proto/document

replace github.com/Oniqq60/task_system_control/gen/proto/task => ../gen/proto/task
