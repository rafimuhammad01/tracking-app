run-tracking:
	go build ./cmd/tracking-service && ./tracking-service -env_file config/development.yaml

run-driver:
	go build ./cmd/driver-service && ./driver-service -env_file config/development.yaml