
lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.45.2 golangci-lint run -v

install:
	go install .