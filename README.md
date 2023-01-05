## Build webmng utility in docker
```bash
docker run --volume="$(pwd):/opt/webmng" webmng-apache-ubuntu make build
```

## Run webmng utility in docker
```bash
docker run --volume="$(pwd):/opt/webmng" webmng-apache-ubuntu ./build/webmng apache version
```
