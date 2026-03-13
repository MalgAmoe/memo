.PHONY: build install clean

FIREWORKS_KEY := $(shell grep FIREWORKS_API_KEY .env 2>/dev/null | cut -d= -f2)
LDFLAGS := -X memo/internal.defaultFireworksKey=$(FIREWORKS_KEY)

build:
	go build -ldflags "$(LDFLAGS)" -o memo ./cmd/memo

install: build
	mkdir -p ~/.local/bin
	cp memo ~/.local/bin/

clean:
	rm -f memo
