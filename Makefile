run-tracking:
	go build ./cmd/tracking-service && ./tracking-service -env_file config/development.yaml