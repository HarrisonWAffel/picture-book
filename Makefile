docker:
	./build/buildDocker.sh

docker-run:
	./build/buildDocker.sh
	./build/runDocker.sh

bin:
	go build -o picture-book

