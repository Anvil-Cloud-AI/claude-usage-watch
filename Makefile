BIN_DIR   ?= $(HOME)/.local/bin
BIN       := $(BIN_DIR)/claude-usage-watch
LABEL     := com.anvilcloud.claude-usage-watch
PLIST     := $(HOME)/Library/LaunchAgents/$(LABEL).plist
LOG       := $(HOME)/Library/Logs/claude-usage-watch.log
THRESHOLD ?= 90
INTERVAL  ?= 5m

.PHONY: build test install uninstall status logs

build:
	go build -o claude-usage-watch .

test:
	go test -race -cover ./...

install:
	mkdir -p $(BIN_DIR) $(HOME)/Library/LaunchAgents
	go build -o $(BIN) .
	sed -e 's|__BIN__|$(BIN)|' -e 's|__THRESHOLD__|$(THRESHOLD)|' \
	    -e 's|__INTERVAL__|$(INTERVAL)|' -e 's|__LOG__|$(LOG)|g' \
	    launchd/$(LABEL).plist.tmpl > $(PLIST)
	-launchctl bootout gui/$$(id -u)/$(LABEL) 2>/dev/null
	launchctl bootstrap gui/$$(id -u) $(PLIST)
	@echo "Installed. Logs: $(LOG)"

uninstall:
	-launchctl bootout gui/$$(id -u)/$(LABEL)
	rm -f $(PLIST) $(BIN)

status:
	launchctl print gui/$$(id -u)/$(LABEL) | grep -E 'state|pid|last exit' || true

logs:
	tail -f $(LOG)
