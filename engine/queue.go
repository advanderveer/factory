package engine

import (
	"context"
	"fmt"

	"github.com/advanderveer/factory/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/pkg/errors"
)

//Q is our local name of our queing interface
type Q sqsiface.SQSAPI

var (
	//AWSQueueRegion determines where our queues are hosted
	AWSQueueRegion = "eu-west-1"

	//AWSQueueAccount determines what account our queues are in
	AWSQueueAccount = "399106104436"

	//QueueNamePrefix makes queues from out stack identifable
	QueueNamePrefix = "factory-node-"
)

//FmtQueueName will return a deterministic queueu name for a node
func FmtQueueName(pk model.NodePK) string {
	return fmt.Sprintf("%s%s", QueueNamePrefix, pk)
}

//FmtQueueURL will setup deterministicly return a queue url
func FmtQueueURL(pk model.NodePK) string {
	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", AWSQueueRegion, AWSQueueAccount, FmtQueueName(pk))
}

//CreateNodeQueue creates a queue based on the nodes primary key
func CreateNodeQueue(ctx context.Context, q Q, pk model.NodePK) (err error) {
	inp := &sqs.CreateQueueInput{}
	inp.SetQueueName(FmtQueueName(pk))
	if _, err := q.CreateQueueWithContext(ctx, inp); err != nil {
		return errors.Wrap(err, "failed to create queue")
	}

	return nil
}

//DeleteNodeQueue deletes a queue based on the primary rimary key
func DeleteNodeQueue(ctx context.Context, q Q, pk model.NodePK) (err error) {
	inp := &sqs.DeleteQueueInput{}
	inp.SetQueueUrl(FmtQueueURL(pk))
	if _, err := q.DeleteQueueWithContext(ctx, inp); err != nil {
		return errors.Wrap(err, "failed to delete queue")
	}

	return nil
}

//NextNodeMessage iterates the node queue by fetching one at a time
func NextNodeMessage(ctx context.Context, q Q, pk model.NodePK) (msg string, err error) {
	inp := &sqs.ReceiveMessageInput{}
	inp.SetQueueUrl(FmtQueueURL(pk))
	inp.SetWaitTimeSeconds(20)
	out := &sqs.ReceiveMessageOutput{}
	if out, err = q.ReceiveMessageWithContext(ctx, inp); err != nil {
		return "", errors.Wrap(err, "failed to receive message")
	}

	if len(out.Messages) < 1 {
		return "", nil
	}

	dinp := &sqs.DeleteMessageInput{}
	dinp.SetQueueUrl(FmtQueueURL(pk))
	dinp.SetReceiptHandle(aws.StringValue(out.Messages[0].ReceiptHandle))
	if _, err = q.DeleteMessageWithContext(ctx, dinp); err != nil {
		return "", errors.Wrap(err, "failed to delete received message")
	}

	return aws.StringValue(out.Messages[0].Body), nil
}

//SendNodeMessage will dispatch a message to the node
func SendNodeMessage(ctx context.Context, q Q, pk model.NodePK, msg string) (err error) {
	inp := &sqs.SendMessageInput{}
	inp.SetQueueUrl(FmtQueueURL(pk))
	inp.SetMessageBody(msg)
	if _, err = q.SendMessage(inp); err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	return nil
}
