default:

gendoc: .venv
	.venv/bin/python3 ci/gendoc.py $(shell grep -l '@module ' *.go) >wiki/Home.md

.venv:
	python -m venv .venv
	.venv/bin/pip install -r ci/requirements.txt

# --- Wiki Updates

pull_wiki:
	git subtree pull --prefix=wiki https://github.com/Luzifer/discord-community.wiki.git master --squash

push_wiki:
	git subtree push --prefix=wiki https://github.com/Luzifer/discord-community.wiki.git master
