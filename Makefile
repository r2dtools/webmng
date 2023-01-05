.PHONY: test
test: test_common test_apache test_nginx

test_common:
	echo "Run common tests"
	go test $(shell go list ./... | grep -v /internal/) -cover

test_apache:
	echo "Run apache tests"
	docker run --volume="$(shell pwd):/opt/webmng" webmng-apache-ubuntu ./scripts/testrun-apache.sh -cover

test_nginx:
	echo "Run nginx tests"
	docker run --volume="$(shell pwd):/opt/webmng" webmng-nginx-ubuntu ./scripts/testrun-nginx.sh -cover

.PHONY: build
build:
	go build -o ./build/webmng -v cmd/main.go

build_apache_ubuntu_image:
	docker build --tag="webmng-apache-ubuntu" -f build/apache/Dockerfile.ubuntu ./

build_nginx_ubuntu_image:
	docker build --tag="webmng-nginx-ubuntu" -f build/nginx/Dockerfile.ubuntu ./

build_all_images: build_apache_ubuntu_image build_nginx_ubuntu_image

clean_all:
	$(shell ./scripts/clean-all.sh)

clean_containers:
	$(shell ./scripts/clean-containers.sh)
