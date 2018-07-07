main: benchmark

clean:
	rm -fr bin

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/benchmark

deploy: clean build
	sls deploy

benchmark: deploy
	sls invoke -f benchmark

destroy:
	sls remove
