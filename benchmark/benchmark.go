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
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	uuid "github.com/satori/go.uuid"
)

var DefaultConfig = &BenchmarkConfig{
	NumWorkers: 10,
	Duration:   4 * time.Minute,
}

func New(name string, c dynamodbiface.DynamoDBAPI, tableName string) *Benchmark {
	return &Benchmark{
		name:      name,
		client:    c,
		cw:        cloudwatch.New(session.Must(session.NewSession())),
		tableName: tableName,
	}
}

type BenchmarkConfig struct {
	NumWorkers int
	Duration   time.Duration
}

type Benchmark struct {
	name      string
	client    dynamodbiface.DynamoDBAPI
	cw        cloudwatchiface.CloudWatchAPI
	tableName string
}

func (b *Benchmark) Run(cfg *BenchmarkConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < cfg.NumWorkers; i++ {
		wg.Add(1)
		go func() {
			b.startWorker(ctx, fmt.Sprintf("%s worker %d", b.name, i))
			wg.Done()
		}()
		time.Sleep(1 * time.Second / time.Duration(cfg.NumWorkers))
	}
	wg.Wait()
}

// startWorker performs one read request per second and on every tenth second
// also a write request. It stops when the passed context is cancelled.
func (b *Benchmark) startWorker(ctx context.Context, name string) {
	log.Printf("Starting %s", name)
	if b.itemExists(ctx, name) {
		log.Printf("Stopping %s, it looks like another one is active", name)
		return
	}

	var i int
	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping %s", name)
			b.cleanup(name)
			return
		case <-time.Tick(1 * time.Second):
			if i%10 == 0 {
				b.writeAccess(ctx, name)
			}
			b.readAccess(ctx, name)
			i++
		}
	}
}

func (b *Benchmark) writeAccess(ctx context.Context, name string) {
	defer b.reportLatency(ctx, "PutItem", time.Now())

	_, err := b.client.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(b.tableName),
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

func (b *Benchmark) readAccess(ctx context.Context, name string) {
	defer b.reportLatency(ctx, "GetItem", time.Now())

	_, err := b.client.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(b.tableName),
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

func (b *Benchmark) itemExists(ctx context.Context, name string) bool {
	out, err := b.client.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(b.tableName),
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

func (b *Benchmark) cleanup(name string) {
	_, err := b.client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(b.tableName),
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

func (b *Benchmark) reportLatency(ctx context.Context, op string, start time.Time) {
	l := time.Now().Sub(start)
	_, err := b.cw.PutMetricDataWithContext(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("DaxBenchmark"),
		MetricData: []*cloudwatch.MetricDatum{{
			Dimensions: []*cloudwatch.Dimension{{
				Name:  aws.String("Backend"),
				Value: aws.String(b.name),
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
