.PHONY: test clean check-podman

NETWORK_NAME = testnetwork
DB_CONTAINER = test-mariadb
APP_CONTAINER = test-readthenburn
DB_PASSWORD = testpass123
AUTH_HEADER = testauth123
AUTH_HEADER ?= testauth123

# Check if podman is available and running
check-podman:
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "❌ Error: Podman is not installed. Please install podman first."; \
		exit 1; \
	fi
	@if ! podman info >/dev/null 2>&1; then \
		echo "❌ Error: Podman daemon is not running or there are permission issues."; \
		exit 1; \
	fi
	@echo "✅ Podman is available and running"

test: check-podman
	@echo "Creating podman network..."
	podman network create $(NETWORK_NAME) || true

	@echo "Starting MariaDB container..."
	podman run -d --name $(DB_CONTAINER) \
		--network $(NETWORK_NAME) \
		-e MYSQL_ROOT_PASSWORD=$(DB_PASSWORD) \
		-e MYSQL_DATABASE=burndb \
		-e MYSQL_USER=burnuser \
		-e MYSQL_PASSWORD=$(DB_PASSWORD) \
		docker.io/library/mariadb:latest

	@echo "Building application container..."
	podman build -t readthenburn .

	@echo "Waiting for MariaDB to be ready..."
	sleep 10

	@echo "Starting application container..."
	podman run -d --name $(APP_CONTAINER) \
		--network $(NETWORK_NAME) \
		-p 8080:8080 \
		-e MYSQL_HOSTNAME=$(DB_CONTAINER) \
		-e MYSQL_DATABASE=burndb \
		-e MYSQL_USERNAME=burnuser \
		-e MYSQL_PASSWORD=$(DB_PASSWORD) \
		-e AUTHHEADER_PASSWORD=$(AUTH_HEADER) \
		-e CORS_HEADER="*" \
		-e SECRET_KEY="7AE49A19B3C844BDB68E460D9224A5D0" \
		readthenburn

	@echo "Running integration tests..."
	./integration_test.sh $(AUTH_HEADER)

clean:
	@echo "Cleaning up containers..."
	podman stop $(APP_CONTAINER) $(DB_CONTAINER) || true
	podman rm $(APP_CONTAINER) $(DB_CONTAINER) || true
	podman network rm $(NETWORK_NAME) || true