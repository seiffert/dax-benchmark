# dax-benchmark

This repository is the code I used to test the performance of a [DAX](https://aws.amazon.com/dynamodb/dax/) cluster compared to a plain [DynamoDB](https://aws.amazon.com/dynamodb/) table.

## Approach

This benchmark code simulates very homogeneous read- and write-traffic against a DAX cluster and a DynamoDB table. To do this, it uses a Lambda function that generates traffic on a DAX cluster and on a DynamoDB table in parallel for 4 minutes. The read- and write-latency of DynamoDB items from both setups can be found in CloudWatch afterwards.

## Run it!

In order to run the benchmark by yourself, follow these steps:

1. [Install Serverless](https://serverless.com/framework/docs/getting-started/)
1. Export AWS credentials on your shell (i.e. `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`)
1. Run `make` in your checkout of this repository