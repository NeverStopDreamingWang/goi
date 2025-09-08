GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOARCH=$(shell go env GOARCH)
GOOS=$(shell go env GOOS)

BASE_PAH := $(shell pwd)
WEB_PATH=$(BASE_PAH)/frontend
SERVER_PATH=$(BASE_PAH)/server

BUILD_PATH = $(BASE_PAH)/build
APP_NAME=example

build_frontend:
	cd $(WEB_PATH) && npm install && npm run build:pro

build_linux:
	cd $(SERVER_PATH) \
    && GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -trimpath -ldflags '-s -w' -o $(BUILD_PATH)/$(APP_NAME)

build_backend_on_darwin:
	cd $(SERVER_PATH) \
    && GOOS=linux GOARCH=amd64 $(GOBUILD) -trimpath -ldflags '-s -w'  -o $(BUILD_PATH)/$(APP_NAME)

upx_bin:
	upx $(BUILD_PATH)/$(APP_NAME)

build_darwin: build_backend_on_darwin upx_bin
