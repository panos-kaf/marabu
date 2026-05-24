BIN=../bin
MARABU=./cmd/marabu

TOPOLOGIES:= standard no-bootstrap bootstrap-only
UIS := cli tui headless
BUILD_TARGETS := $(foreach n,$(TOPOLOGIES),$(foreach u,$(UIS),$(n)-$(u)))

.PHONY: all build tests run

all: build tests

# This automatically depends on all 9 combinations
build: $(BUILD_TARGETS)

# $* captures the wildcard part (e.g., "no-bootstrap").
# $(subst -,_,$*) swaps the dash for an underscore for your Go tags!

%-cli:
	go build -tags "$(subst -,_,$*) cli" -o $(BIN)/marabu-$*-cli $(MARABU)

%-tui:
	go build -tags "$(subst -,_,$*) tui" -o $(BIN)/marabu-$*-tui $(MARABU)

%-headless:
	go build -tags "$(subst -,_,$*) headless" -o $(BIN)/marabu-$*-headless $(MARABU)


# --- Testing & Execution ---

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
