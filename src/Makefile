BIN=../bin
MARABU=./cmd/marabu

all: build tests

build: standard-cli standard-headless bootstrap-only-cli bootstrap-only-headless no-bootstrap-cli no-bootstrap-headless 

standard-cli:
	go build -tags "standard cli" -o $(BIN)/marabu-standard-cli $(MARABU)

standard-headless:
	go build -tags "standard headless" -o $(BIN)/marabu-standard-headless $(MARABU)

no-bootstrap-cli:
	go build -tags "no_bootstrap cli" -o $(BIN)/marabu-no-bootstrap-cli $(MARABU)

no-bootstrap-headless:
	go build -tags "no_bootstrap headless" -o $(BIN)/marabu-no-bootstrap-headless $(MARABU)

bootstrap-only-cli:
	go build -tags "bootstrap_only cli" -o $(BIN)/marabu-bootstrap-only-cli $(MARABU)

bootstrap-only-headless:
	go build -tags "bootstrap_only headless" -o $(BIN)/marabu-bootstrap-only-headless $(MARABU)

tests: test-obj test-obj-simple test-pset2 test-utxo

test-obj-simple:
	go build -o $(BIN)/objectSimple ./cmd/testing/objectSimple

test-obj:
	go build -o $(BIN)/object ./cmd/testing/object

test-pset2:
	go build -o $(BIN)/pset2grader ./cmd/testing/pset2

test-utxo:
	go build -o $(BIN)/utxo ./cmd/testing/utxo

run:
	go run $(MARABU)
