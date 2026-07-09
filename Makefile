TARGETBIN=eim-skill
BUILD_ROOT=$(PWD)

.PHONY: all install clean

all: $(TARGETBIN)

$(TARGETBIN):
	@gofmt -l -w $(BUILD_ROOT)/
	go build -ldflags "-w -s" -o $@ .

install: all
	@mkdir -p out
	@cp -a conf $(TARGETBIN) out/
	@echo "[Done]"

clean:
	@rm -rf $(TARGETBIN) *.log *.tar.gz
	@echo "[clean Done]"
