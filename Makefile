main: benchmark

clean:
	rm -fr bin

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/benchmark-dynamodb cmd/benchmark-dynamodb/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/benchmark-dax cmd/benchmark-dax/main.go

deploy: clean build
	sls deploy

benchmark: deploy
	sls invoke -f benchmark-dynamodb
	sls invoke -f benchmark-dax

destroy:
	sls remove