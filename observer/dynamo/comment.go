package dynamo

import (
	"github.com/a-h/watchman/observer/data"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func NewCommentStore(region, tableName string) (store *CommentStore, err error) {
	conf := &aws.Config{
		Region: aws.String(region),
	}
	sess, err := session.NewSession(conf)
	if err != nil {
		return
	}
	store = &CommentStore{
		Client:    dynamodb.New(sess),
		TableName: aws.String(tableName),
	}
	return
}

type CommentStore struct {
	Client    *dynamodb.DynamoDB
	TableName *string
}

func (store CommentStore) Put(r data.Comment) (updated bool, err error) {
	item, err := dynamodbattribute.MarshalMap(r)
	if err != nil {
		return
	}
	rv, err := store.Client.PutItem(&dynamodb.PutItemInput{
		TableName:    store.TableName,
		Item:         item,
		ReturnValues: aws.String(dynamodb.ReturnValueUpdatedOld),
	})
	_, updated = rv.Attributes["url"]
	return
}

// Get retrieves data from DynamoDB.
func (store CommentStore) Get(url string) (r data.Comment, ok bool, err error) {
	gio, err := store.Client.GetItem(&dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"url": {
				S: aws.String(url),
			},
		},
		TableName: store.TableName,
	})
	if err != nil {
		return
	}
	err = dynamodbattribute.UnmarshalMap(gio.Item, &r)
	ok = r.URL != ""
	return
}
