.PHONY: build install clean

build:
	go build -o memo ./cmd/memo

install: build
	mkdir -p ~/.local/bin
	cp memo ~/.local/bin/

clean:
	rm -f memo
