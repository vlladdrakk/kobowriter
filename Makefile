PWD=$(shell pwd)

.PHONY: build
build:
	CGO_ENABLED=1 GOARCH=arm GOOS=linux CC=${CROSS_TC}-gcc CXX=${CROSS_TC}-g++ go build -o ./build/kobowriter

docker_build:
	docker build . -t kobo-builder

docker:
	docker run --rm -v ${PWD}:/home/ubuntu/app kobo-builder bash -c "cd /home/ubuntu/app && go get && make"

docker_run:
	docker run --rm -v ${PWD}:/home/ubuntu/app kobo-builder bash -c "cd /home/ubuntu/app && exec $@"