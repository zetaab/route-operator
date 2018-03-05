OPERATOR_NAME  := f5-route-operator
IMAGE := foobar.com/devops/$(OPERATOR_NAME)/public/$(OPERATOR_NAME)
.PHONY: install_deps build build-image

install_deps:
	go get ./...

build:
	rm -rf bin/$(OPERATOR_NAME)
	go build -v -i -o bin/$(OPERATOR_NAME) ./pkg/controller

build-image:
	rm -rf bin/linux/
	mkdir -p bin/linux
	GOOS=linux GOARCH=amd64 go build -v -i -o bin/linux/$(OPERATOR_NAME) ./cmd
	docker build -t $(IMAGE):latest .