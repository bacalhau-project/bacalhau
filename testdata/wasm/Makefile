%.wat: %.wasm
	wasm2wat $^ > $@

%.wasm:
	pushd $(dir $@) && \
	cargo build --target wasm32-wasi --release && \
	popd && \
	cp $(dir $@)/target/wasm32-wasi/release/*.wasm $@

%.csv: %.sparql
	curl -G https://query.wikidata.org/sparql \
		--data-urlencode "query=$$(cat $^)" \
		-H "Accept: text/csv" -o $@

all: $(patsubst ./%,%/main.wasm,$(shell find . -type d -depth 1))
