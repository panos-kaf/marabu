SRC=./src

all:
	make -C $(SRC) all

build:
	make -C $(SRC) build

standard-cli:
	make -C $(SRC) standard-cli

standard-headless:
	make -C $(SRC) standard-headless

no-bootstrap-headless:
	make -C $(SRC) no-bootstrap-headless

no-bootstrap-cli:
	make -C $(SRC) no-bootstrap-cli

tests:
	make -C $(SRC) tests

test-obj:
	make -C $(SRC) test-obj

test-obj-simple:
	make -C $(SRC) test-obj-simple

test-pset2:
	make -C $(SRC) test-pset2

test-utxo:
	mace -C $(SRC) test-utxo

run:
	make -C $(SRC) run

clean:
	$(RM) bin/*
	$(RM) logs/*.log

rebuild: clean build
