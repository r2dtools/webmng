.PHONY: test
test:
	docker run --volume="$(shell pwd):/opt/webmng" webmng-test -cover

.PHONY: build
build:
	go build -o ./build/webmng -v cmd/main.go

build_test_image:
	docker build --no-cache --tag="webmng-test" -f Dockerfile.test ./

clean:
	$(shell ./scripts/clean.sh)
