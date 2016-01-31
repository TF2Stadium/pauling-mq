docker:
	go build -ldflags "-linkmode external -extldflags -static" -v -o pauling-mq
	docker build -t tf2stadium/pauling-mq .
