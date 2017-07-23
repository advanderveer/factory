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

//ScheduleMsg is used for the scheduling queue
type ScheduleMsg struct {
	PoolID string `json:"pool_id"`
	Size   int64  `json:"size"`
}

//RunMsg is the msg send to nodes
type RunMsg struct {
	Size int64 `json:"size"`
}

var (
	//AWSQueueRegion determines where our queues are hosted
	AWSQueueRegion = "eu-west-1"

	//AWSQueueAccount determines what account our queues are in
	AWSQueueAccount = "399106104436"

	//ScheduleQueueName is the name of the queue that contains schedule requests
	ScheduleQueueName = "factory-scheduling"

	//NodeQueuePrefix makes queues from out stack identifable
	NodeQueuePrefix = "factory-node-"
)

//FmtQueueName will return a deterministic queueu name for a node
func FmtQueueName(pk model.NodePK) string {
	return fmt.Sprintf("%s%s", NodeQueuePrefix, pk)
}

//FmtQueueURL will setup deterministicly return a queue url
func FmtQueueURL(name string) string {
	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", AWSQueueRegion, AWSQueueAccount, name)
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
	inp.SetQueueUrl(FmtQueueURL(FmtQueueName(pk)))
	if _, err := q.DeleteQueueWithContext(ctx, inp); err != nil {
		return errors.Wrap(err, "failed to delete queue")
	}

	return nil
}

//NextNodeMessage iterates the node queue by fetching one at a time
func NextNodeMessage(ctx context.Context, q Q, pk model.NodePK) (msg string, err error) {
	inp := &sqs.ReceiveMessageInput{}
	inp.SetQueueUrl(FmtQueueURL(FmtQueueName(pk)))
	inp.SetWaitTimeSeconds(20)
	out := &sqs.ReceiveMessageOutput{}
	if out, err = q.ReceiveMessageWithContext(ctx, inp); err != nil {
		return "", errors.Wrap(err, "failed to receive message")
	}

	if len(out.Messages) < 1 {
		return "", nil
	}

	dinp := &sqs.DeleteMessageInput{}
	dinp.SetQueueUrl(FmtQueueURL(FmtQueueName(pk)))
	dinp.SetReceiptHandle(aws.StringValue(out.Messages[0].ReceiptHandle))
	if _, err = q.DeleteMessageWithContext(ctx, dinp); err != nil {
		return "", errors.Wrap(err, "failed to delete received message")
	}

	return aws.StringValue(out.Messages[0].Body), nil
}

//NextScheduleMessage iterates the schedule queue by fetching one at a time
func NextScheduleMessage(ctx context.Context, q Q, handler func(msg string) bool) (err error) {
	inp := &sqs.ReceiveMessageInput{}
	inp.SetQueueUrl(FmtQueueURL(ScheduleQueueName))
	inp.SetWaitTimeSeconds(20)
	out := &sqs.ReceiveMessageOutput{}
	if out, err = q.ReceiveMessageWithContext(ctx, inp); err != nil {
		return errors.Wrap(err, "failed to receive message")
	}

	if len(out.Messages) < 1 {
		return nil
	}

	if handler(aws.StringValue(out.Messages[0].Body)) {
		dinp := &sqs.DeleteMessageInput{}
		dinp.SetQueueUrl(FmtQueueURL(ScheduleQueueName))
		dinp.SetReceiptHandle(aws.StringValue(out.Messages[0].ReceiptHandle))
		if _, err = q.DeleteMessageWithContext(ctx, dinp); err != nil {
			return errors.Wrap(err, "failed to delete received message")
		}
	}

	return nil
}

//SendScheduleMessage will dispatch a message to the node
func SendScheduleMessage(ctx context.Context, q Q, msg string) (err error) {
	inp := &sqs.SendMessageInput{}
	inp.SetQueueUrl(FmtQueueURL(ScheduleQueueName))
	inp.SetMessageBody(msg)
	if _, err = q.SendMessageWithContext(ctx, inp); err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	return nil
}

//SendNodeMessage will dispatch a message to the node
func SendNodeMessage(ctx context.Context, q Q, pk model.NodePK, msg string) (err error) {
	inp := &sqs.SendMessageInput{}
	inp.SetQueueUrl(FmtQueueURL(FmtQueueName(pk)))
	inp.SetMessageBody(msg)
	if _, err = q.SendMessageWithContext(ctx, inp); err != nil {
		return errors.Wrap(err, "failed to send message")
	}

	return nil
}
