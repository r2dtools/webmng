.PHONY: test
test: test_common test_apache test_nginx

test_common:
	echo "Run common tests"
	go test $(shell go list ./... | grep -v /internal/) -cover

test_apache:
	echo "Run apache tests"
	docker run --volume="$(shell pwd):/opt/webmng" webmng-apache-test -cover

test_nginx:
	echo "Run nginx tests"
	docker run --volume="$(shell pwd):/opt/webmng" webmng-nginx-test -cover

.PHONY: build
build:
	go build -o ./build/webmng -v cmd/main.go

build_apache_test_image:
	docker build --no-cache --tag="webmng-apache-test" -f Dockerfile.apache.test ./

build_nginx_test_image:
	docker build --no-cache --tag="webmng-nginx-test" -f Dockerfile.nginx.test ./

clean:
	$(shell ./scripts/testclean.sh)
