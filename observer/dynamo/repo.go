package dynamo

import (
	"fmt"
	"time"

	"github.com/a-h/watchman/observer/data"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func NewRepoStore(region, tableName string) (store *RepoStore, err error) {
	conf := &aws.Config{
		Region: aws.String(region),
	}
	sess, err := session.NewSession(conf)
	if err != nil {
		return
	}
	store = &RepoStore{
		Client:    dynamodb.New(sess),
		TableName: aws.String(tableName),
	}
	return
}

type RepoStore struct {
	Client    *dynamodb.DynamoDB
	TableName *string
}

func (store RepoStore) Add(service, url, usedByUrl string) (err error) {
	u := expression.Add(expression.Name("usedByUrls"), expression.Value(usedByUrl))
	expr, err := expression.NewBuilder().
		WithUpdate(u).
		Build()
	if err != nil {
		return
	}
	_, err = store.Client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: store.TableName,
		Key: map[string]*dynamodb.AttributeValue{
			"service": &dynamodb.AttributeValue{
				S: aws.String(service),
			},
			"url": &dynamodb.AttributeValue{
				S: aws.String(url),
			},
		},
		ExpressionAttributeValues: expr.Values(),
		ExpressionAttributeNames:  expr.Names(),
		UpdateExpression:          expr.Update(),
		ReturnValues:              aws.String(dynamodb.ReturnValueNone),
	})
	return
}

func (store RepoStore) Update(service, url string, lastUpdated time.Time) (err error) {
	u := expression.Set(expression.Name("lastUpdated"), expression.Value(lastUpdated))
	expr, err := expression.NewBuilder().
		WithUpdate(u).
		Build()
	if err != nil {
		return
	}
	_, err = store.Client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: store.TableName,
		Key: map[string]*dynamodb.AttributeValue{
			"service": &dynamodb.AttributeValue{
				S: aws.String(service),
			},
			"url": &dynamodb.AttributeValue{
				S: aws.String(url),
			},
		},
		ExpressionAttributeValues: expr.Values(),
		ExpressionAttributeNames:  expr.Names(),
		UpdateExpression:          expr.Update(),
		ReturnValues:              aws.String(dynamodb.ReturnValueNone),
	})
	return
}

// Query by the service name.
func (store RepoStore) Query(service string) (repoData []data.Repository, ok bool, err error) {
	q := expression.Key("service").Equal(expression.Value(service))

	expr, err := expression.NewBuilder().
		WithKeyCondition(q).
		Build()
	if err != nil {
		err = fmt.Errorf("RepoStore.Query: failed to build query: %v", err)
		return
	}

	qi := &dynamodb.QueryInput{
		TableName:                 store.TableName,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
	}

	var pageErr error
	err = store.Client.QueryPages(qi, func(page *dynamodb.QueryOutput, lastPage bool) bool {
		var dataPage []data.Repository
		pageErr = dynamodbattribute.UnmarshalListOfMaps(page.Items, &dataPage)
		if pageErr != nil {
			return true
		}
		repoData = append(repoData, dataPage...)
		return len(page.LastEvaluatedKey) == 0
	})
	if err != nil {
		err = fmt.Errorf("RepoStore.Get: failed to query pages: %v", err)
		return
	}
	if pageErr != nil {
		err = fmt.Errorf("RepoStore.Get: failed to unmarshal maps: %v", pageErr)
		return
	}

	ok = len(repoData) > 0
	return
}
