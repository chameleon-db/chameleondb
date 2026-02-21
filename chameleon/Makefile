.PHONY: build test clean run-parse run-validate test-integration test-all docker-up docker-down wait-db build-rust

# Env variables for tests
TEST_DB_HOST ?= localhost
TEST_DB_PORT ?= 5433
TEST_DB_NAME ?= chameleon_test
TEST_DB_USER ?= postgres
TEST_DB_PASS ?= postgres

export CHAMELEON_TEST_DB_HOST=$(TEST_DB_HOST)
export CHAMELEON_TEST_DB_PORT=$(TEST_DB_PORT)
export CHAMELEON_TEST_DB_NAME=$(TEST_DB_NAME)
export CHAMELEON_TEST_DB_USER=$(TEST_DB_USER)
export CHAMELEON_TEST_DB_PASS=$(TEST_DB_PASS)

# Paths
RUST_LIB_DIR := $(shell pwd)/../chameleon-core/target/release
RUST_INCLUDE_DIR := $(shell pwd)/../chameleon-core/include
CGO_TEST_LDFLAGS := -L$(RUST_LIB_DIR) -lchameleon_core
CGO_TEST_CFLAGS := -I$(RUST_INCLUDE_DIR)
UNIT_TEST_PACKAGES := $(shell go list ./... | grep -v '/tests/integration')

# Build Rust core first
build-rust:
	cd ../chameleon-core && cargo build --release

# Build Go binary with explicit library path
build: build-rust
	CGO_LDFLAGS="-L$(RUST_LIB_DIR) -lchameleon_core -Wl,-rpath,$(RUST_LIB_DIR)" \
	CGO_CFLAGS="-I$(RUST_INCLUDE_DIR)" \
	go build -o bin/chameleon ./cmd/chameleon

# Run tests
test: build-rust
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" \
	CGO_CFLAGS="$(CGO_TEST_CFLAGS)" \
	LD_LIBRARY_PATH=$(RUST_LIB_DIR) \
	go test -v $(UNIT_TEST_PACKAGES) -count=1

test-integration:
	@echo "Building Rust library..."
	cd ../chameleon-core && cargo build --release

	@echo "Starting test containers..."
	$(MAKE) docker-up

	@echo "Waiting for PostgreSQL..."
	$(MAKE) wait-db

	@echo "Running integration tests..."
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" \
	CGO_CFLAGS="$(CGO_TEST_CFLAGS)" \
	LD_LIBRARY_PATH=$(RUST_LIB_DIR) \
	go test -v ./tests/integration/... -timeout 60s -count=1 || \
	($(MAKE) docker-down && exit 1)

	@echo "Stopping test containers..."
	$(MAKE) docker-down


test-all:
	@echo "Building Rust library..."
	cd ../chameleon-core && cargo build --release

	@echo "Starting test containers..."
	$(MAKE) docker-up

	@echo "Waiting for PostgreSQL..."
	$(MAKE) wait-db

	@echo "Running all tests (unit + integration)..."
	CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)" \
	CGO_CFLAGS="$(CGO_TEST_CFLAGS)" \
	LD_LIBRARY_PATH=$(RUST_LIB_DIR) \
	go test -v ./... -timeout 60s -count=1 || \
	($(MAKE) docker-down && exit 1)

	@echo "Stopping test containers..."
	$(MAKE) docker-down

docker-up:
	docker compose -f docker-compose.test.yml up -d

docker-down:
	docker compose -f docker-compose.test.yml down -v

wait-db:
	@for i in $$(seq 1 30); do \
		docker compose -f docker-compose.test.yml exec -T postgres \
			pg_isready -U $(TEST_DB_USER) -d $(TEST_DB_NAME) >/dev/null 2>&1 && break; \
		if [ $$i -eq 30 ]; then \
			echo "PostgreSQL did not become ready in time"; \
			$(MAKE) docker-down; \
			exit 1; \
		fi; \
		sleep 1; \
	done

# Clean builds
clean:
	rm -rf bin/
	cd ../chameleon-core && cargo clean

# Run examples (need LD_LIBRARY_PATH at runtime)
run-parse: build
	LD_LIBRARY_PATH=$(RUST_LIB_DIR) ./bin/chameleon parse ../examples/basic_schema.cham

run-validate: build
	LD_LIBRARY_PATH=$(RUST_LIB_DIR) ./bin/chameleon validate ../examples/basic_schema.cham

# Install locally
install: build
	cp bin/chameleon $(GOPATH)/bin/