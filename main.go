package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/seiffert/dax-benchmark/benchmark"

	"github.com/aws/aws-dax-go/dax"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func Handler() error {
	endpoint := os.Getenv("DAX_ENDPOINT")
	log.Printf("Using DAX discovery endpoint %q", endpoint)

	cfg := dax.DefaultConfig()
	cfg.HostPorts = []string{endpoint}
	cfg.Region = os.Getenv("AWS_REGION")
	cfg.LogLevel = aws.LogDebug
	daxClient, err := dax.New(cfg)
	if err != nil {
		return fmt.Errorf("unable to initialize dax client %v", err)
	}

	// disable retries in order to generate a predictable amount of requests
	dynamoDBClient := dynamodb.New(session.Must(session.NewSession(aws.NewConfig().WithMaxRetries(0))))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		benchmark.New("DAX", daxClient, os.Getenv("DYNAMODB_TABLE_DAX")).Run(benchmark.DefaultConfig)
		wg.Done()
	}()
	go func() {
		benchmark.New("DynamoDB", dynamoDBClient, os.Getenv("DYNAMODB_TABLE_NODAX")).Run(benchmark.DefaultConfig)
		wg.Done()
	}()
	wg.Wait()

	return nil
}

func main() {
	lambda.Start(Handler)
}
