BINARY := yabai-stateful-workspaces
BUILD_DIR := ./dist

.PHONY: all build clean run install

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) .
	@echo "✓  built $(BUILD_DIR)/$(BINARY)"

run: build
	$(BUILD_DIR)/$(BINARY)

install: build
	install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "✓  installed to /usr/local/bin/$(BINARY)"

clean:
	rm -rf $(BUILD_DIR)
	rm -f /tmp/yabai-stateful-workspaces.fifo
