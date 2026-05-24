SRC=./src

.PHONY: default clean rebuild

default: 
	$(MAKE) -C $(SRC) standard-tui 

clean:
	$(RM) bin/*
	$(RM) logs/*.log

rebuild: clean 
	$(MAKE) -C $(SRC) build

# Catch-all rule: Forwards any command (like 'make tests') to the src Makefile!
%:
	$(MAKE) -C $(SRC) $@
