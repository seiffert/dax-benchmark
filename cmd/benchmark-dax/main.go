package main

import (
	"fmt"
	"log"
	"os"

	"github.com/seiffert/dax-benchmark/benchmark"

	"github.com/aws/aws-dax-go/dax"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
)

type Response struct {
	Message string `json:"message"`
}

func Handler() (Response, error) {
	endpoint := os.Getenv("DAX_ENDPOINT")
	log.Printf("Using DAX discovery endpoint %q", endpoint)

	cfg := dax.DefaultConfig()
	cfg.HostPorts = []string{endpoint}
	cfg.Region = "eu-west-1"
	cfg.LogLevel = aws.LogDebug
	client, err := dax.New(cfg)
	if err != nil {
		panic(fmt.Errorf("unable to initialize dax client %v", err))
	}

	benchmark.New("benchmark-dax", client).Run()

	return Response{
		Message: "I'm done!",
	}, nil
}

func main() {
	lambda.Start(Handler)
}
