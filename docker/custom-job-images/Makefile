VERSION ?= 0.1

local: python-local duckdb-local

build: python duckdb

python: 
	@$(MAKE) -C python build 

python-local: 
	@$(MAKE) -C python local

duckdb: 
	@$(MAKE) -C duckdb build 

duckdb-local: 
	@$(MAKE) -C duckdb local	

.PHONY: local python duckdb


python-test:
	docker run --rm -it  -v $(shell pwd)/python/test/single-file:/code  bacalhauproject/exec-python-3.11:0.1  python  /build/launcher.py -- python hello.py
	docker run --rm -it  -v $(shell pwd)/python/test/multi-file-reqtxt:/code  bacalhauproject/exec-python-3.11:0.1  python  /build/launcher.py -- python main.py
	docker run --rm -it  -v $(shell pwd)/python/test/multi-file-poetry:/code  bacalhauproject/exec-python-3.11:0.1  python  /build/launcher.py -- poetry run mfp