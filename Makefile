BINARY  := tg-alerts
MODULE  := github.com/heliofernandes404/tg-alerts
REPO    := HelioFernandes404/sf-telegram-cli

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

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

INSTALL_DIR ?= $(HOME)/.local/bin

install:
	@mkdir -p $(INSTALL_DIR); \
	ARCH=$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/'); \
	URL=$$(curl -sf https://api.github.com/repos/$(REPO)/releases/latest \
	  | grep "browser_download_url" \
	  | grep "linux_$${ARCH}.tar.gz" \
	  | cut -d '"' -f 4); \
	if [ -z "$$URL" ]; then echo "Error: no release found for linux/$$ARCH" >&2; exit 1; fi; \
	echo "Downloading $$URL ..."; \
	curl -fL "$$URL" | tar -xz -C $(INSTALL_DIR) $(BINARY); \
	echo "Installed $(BINARY) to $(INSTALL_DIR)"
