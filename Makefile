SHELL := /bin/bash
APP_DIR := app
WAILS := wails

.PHONY: mac linux windows all clean build doctor

build:
	cd $(APP_DIR) && $(WAILS) build -clean

mac:
	cd $(APP_DIR) && $(WAILS) build -clean -platform darwin/amd64,darwin/arm64

linux:
	cd $(APP_DIR) && $(WAILS) build -clean -platform linux/amd64,linux/arm64

windows:
	cd $(APP_DIR) && $(WAILS) build -clean -platform windows/amd64,windows/arm64 -nsis -webview2 download

all: mac linux windows

clean:
	cd $(APP_DIR) && rm -rf build/bin

doctor:
	cd $(APP_DIR) && $(WAILS) doctor

