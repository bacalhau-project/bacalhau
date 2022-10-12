%.wat: %.wasm
	wasm2wat $^ > $@

%.wasm:
	pushd $(dir $@) && \
	cargo build --target wasm32-wasi --release && \
	popd && \
	cp $(dir $@)/target/wasm32-wasi/release/*.wasm $@

all: $(patsubst ./%,%/main.wasm,$(shell find . -type d -depth 1))
