.PHONY: all build start run cli watch simulate docker test clean help

all: build

help:
	@./pulsepoll help

build:
	@./pulsepoll build

start:
	@./pulsepoll start

run:
	@./pulsepoll start

cli:
	@./pulsepoll cli

watch:
	@./pulsepoll watch $(ID)

simulate:
	@./pulsepoll simulate $(ID) $(VOTES)

docker:
	@./pulsepoll docker

test:
	@./pulsepoll test

clean:
	@rm -f pulsepoll-server pulsepoll-cli backend/pulsepoll-server backend/pulsepoll-cli
	@rm -rf frontend/dist
	@echo "Cleaned build artifacts."
