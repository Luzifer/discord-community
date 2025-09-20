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
	.venv/bin/python3 ci/gendoc.py $(shell grep -l '@module ' *.go) >wiki/Home.md
	git add wiki/Home.md

.venv:
	python -m venv .venv
	.venv/bin/pip install -r ci/requirements.txt

# --- Wiki Updates

pull_wiki:
	git subtree pull --prefix=wiki https://github.com/Luzifer/discord-community.wiki.git master --squash

push_wiki:
	git subtree push --prefix=wiki https://github.com/Luzifer/discord-community.wiki.git master

# --- Local dev

auto-hook-pre-commit: gendoc

ifeq ($(BRANCH), master)
auto-hook-post-push: push_wiki
endif
