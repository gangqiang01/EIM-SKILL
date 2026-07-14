TARGETBIN=eim-skill
BUILD_ROOT=$(PWD)
MODULE=github.com/gangqiang01/EIM-SKILL
DIST_DIR=dist

VERSION := $(shell cat VERSION 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILDDATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -w -s \
	-X '$(MODULE)/cmd.version=$(VERSION)' \
	-X '$(MODULE)/cmd.commit=$(COMMIT)' \
	-X '$(MODULE)/cmd.buildDate=$(BUILDDATE)'

.PHONY: all install clean cross linux-amd64 linux-arm64 windows-amd64 package

all: $(TARGETBIN)

$(TARGETBIN):
	@gofmt -l -w $(BUILD_ROOT)/cmd $(BUILD_ROOT)/internal $(BUILD_ROOT)/main.go
	go build -ldflags "$(LDFLAGS)" -o $@ .

install: all
	@mkdir -p out
	@cp -a conf $(TARGETBIN) out/
	@echo "[Done]"

# Cross-compiled release binaries, written to dist/.
linux-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/eim-skill .

linux-arm64:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/eim-skill-arm64 .

windows-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/eim-skill.exe .

cross: linux-amd64 linux-arm64 windows-amd64

# Cross-compile every platform, then bundle README.md, README-zh.md,
# skills/, conf/config.yaml.example, and the compiled binaries into
# DeviceOn-EIM-SKILL_<version>.tar.gz.
package: cross
	@./scripts/package.sh $(VERSION)

clean:
	@rm -rf $(TARGETBIN) *.log *.tar.gz $(DIST_DIR)
	@echo "[clean Done]"
