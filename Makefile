BRANCH:=$(shell git branch --show-current)

default:

lint:
	golangci-lint run --timeout=5m

publish:
	bash ./ci/build.sh

test:
	go test -cover -v .

# --- Documentation

gendoc: .venv
	.venv/bin/python3 ci/gendoc.py $(shell grep -l '@module ' pkg/modules/*/*.go) >wiki/Home.md

.venv:
	python -m venv .venv
	.venv/bin/pip install -r ci/requirements.txt
