package sender

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

// SNS sends an sns message
type SNS struct {
	Region string
	Topic  string
}

// NewSNS constructs a new SNS sender.
func NewSNS(region, topic string) SNS {
	return SNS{
		Region: region,
		Topic:  topic,
	}
}

// Send publishes an sns message to the configured topic.
func (pn SNS) Send(subject string, v interface{}) (err error) {
	cfg := &aws.Config{
		Region: aws.String(pn.Region),
	}
	payload, err := json.Marshal(v)
	if err != nil {
		return
	}
	svc := sns.New(session.New(cfg))
	params := &sns.PublishInput{
		Subject:  aws.String(subject),
		Message:  aws.String(string(payload)),
		TopicArn: aws.String(pn.Topic),
	}
	_, err = svc.Publish(params)
	return err
}
