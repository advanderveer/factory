package model

import "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

//DB type is our db interface
type DB dynamodbiface.DynamoDBAPI
