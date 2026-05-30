BINARY  := tg-alerts
MODULE  := github.com/heliofernandes404/tg-alerts

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%d)

LDFLAGS := -s -w \
	-X $(MODULE)/cmd.version=$(VERSION) \
	-X $(MODULE)/cmd.commit=$(COMMIT) \
	-X $(MODULE)/cmd.buildDate=$(DATE)

.PHONY: build clean release install

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

clean:
	rm -f $(BINARY)

release:
ifndef VERSION_ARG
	$(error VERSION_ARG is required. Usage: make release VERSION_ARG=1.0.0)
endif
	git tag v$(VERSION_ARG)
	git push origin v$(VERSION_ARG)

install:
	@ARCH=$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/'); \
	URL=$$(curl -sf https://api.github.com/repos/heliofernandes404/tg-alerts/releases/latest \
	  | grep "browser_download_url" \
	  | grep "linux_$${ARCH}.tar.gz" \
	  | cut -d '"' -f 4); \
	if [ -z "$$URL" ]; then echo "Error: no release found for linux/$$ARCH" >&2; exit 1; fi; \
	echo "Downloading $$URL ..."; \
	curl -fL "$$URL" | sudo tar -xz -C /usr/local/bin $(BINARY); \
	echo "Installed $(BINARY) to /usr/local/bin"
