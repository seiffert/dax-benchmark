package benchmark

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	uuid "github.com/satori/go.uuid"
)

const (
	numWorkers        = 10
	tableName         = "dax-benchmark-table"
	benchmarkDuration = 4 * time.Minute
)

var (
	client       dynamodbiface.DynamoDBAPI
	cw           = cloudwatch.New(session.Must(session.NewSession()))
	functionName string
)

type Response struct {
	Message string `json:"message"`
}

func Benchmark(fn string, c dynamodbiface.DynamoDBAPI) (Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), benchmarkDuration)
	defer cancel()

	client = c
	functionName = fn

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			worker(ctx, fmt.Sprintf("%s worker %d", fn, i))
			wg.Done()
		}()
		time.Sleep(1 * time.Second / numWorkers)
	}
	wg.Wait()

	return Response{
		Message: "I'm done!",
	}, nil
}

// worker performs one read request per second and on every tenth second also
// a write request. It stops when the passed context is cancelled.
func worker(ctx context.Context, name string) {
	log.Printf("Starting %s", name)
	if itemExists(ctx, name) {
		log.Printf("Stopping %s, it looks like another one is active", name)
		return
	}

	var i int
	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping %s", name)
			cleanup(name)
			return
		case <-time.Tick(1 * time.Second):
			if i%10 == 0 {
				writeAccess(ctx, name)
			}
			readAccess(ctx, name)
			i++
		}
	}
}

func writeAccess(ctx context.Context, name string) {
	defer reportLatency(ctx, "PutItem", time.Now())

	_, err := client.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
			"uuid": {
				S: aws.String(uuid.NewV4().String()),
			},
		},
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			err = fmt.Errorf("%s: %s", awsErr.Message(), awsErr.OrigErr())
		}
		log.Printf("Could not put item: %s", err)
	}
}

func readAccess(ctx context.Context, name string) {
	defer reportLatency(ctx, "GetItem", time.Now())

	_, err := client.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
		},
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			err = fmt.Errorf("%s: %s", awsErr.Message(), awsErr.OrigErr())
		}
		log.Printf("Could not get item: %s", err)
	}
}

func itemExists(ctx context.Context, name string) bool {
	out, err := client.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
		},
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			err = fmt.Errorf("%s: %s", awsErr.Message(), awsErr.OrigErr())
		}
		log.Printf("Could not check if item exists: %s", err)
	}
	return out.Item != nil
}

func cleanup(name string) {
	_, err := client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
		},
	})
	if err != nil {
		log.Printf("Could not delete item: %s", err)
	}
}

func reportLatency(ctx context.Context, op string, start time.Time) {
	l := time.Now().Sub(start)
	_, err := cw.PutMetricDataWithContext(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("DaxBenchmark"),
		MetricData: []*cloudwatch.MetricDatum{{
			Dimensions: []*cloudwatch.Dimension{{
				Name:  aws.String("FunctionName"),
				Value: aws.String(functionName),
			}, {
				Name:  aws.String("Operation"),
				Value: aws.String(op),
			}},
			MetricName: aws.String("Latency"),
			Timestamp:  aws.Time(time.Now()),
			Unit:       aws.String("Milliseconds"),
			Value:      aws.Float64(float64(1000 * l.Seconds())),
		}},
	})
	if err != nil {
		log.Printf("Could not write metric data: %s", err)
	}
}
