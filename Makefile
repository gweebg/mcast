# Variables
BUILD_FOLDER := build
CMD_FILES := $(wildcard cmd/*.go)
BINARIES := $(patsubst cmd/%.go,$(BUILD_FOLDER)/%.run,$(CMD_FILES))

# Build targets
all: $(BINARIES)

$(BUILD_FOLDER)/%.run: cmd/%.go
	@mkdir -p $(BUILD_FOLDER)
	@echo "Building $@"
	@go build -o $@ $<

clean:
	@rm -rf $(BUILD_FOLDER)

.PHONY: all clean
