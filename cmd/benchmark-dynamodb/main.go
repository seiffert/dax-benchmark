package main

import (
	"github.com/seiffert/dax-benchmark/benchmark"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type Response struct {
	Message string `json:"message"`
}

func Handler() (Response, error) {
	benchmark.Benchmark("benchmark-dynamodb", dynamodb.New(session.Must(session.NewSession(aws.NewConfig().WithMaxRetries(0)))))

	return Response{
		Message: "I'm done!",
	}, nil
}

func main() {
	lambda.Start(Handler)
}
