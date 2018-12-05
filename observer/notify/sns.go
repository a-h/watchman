package notify

import (
	"encoding/json"
	"fmt"

	"github.com/a-h/watchman/observer/data"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

// NewSNS creates a new SNS notifier.
func NewSNS(topicARN string) SNS {
	return SNS{
		TopicARN: topicARN,
	}
}

// SNS notifier.
type SNS struct {
	TopicARN string
}

// Notify via SNS.
func (s SNS) Notify(riss data.RepositoryIssue) error {
	subject := fmt.Sprintf("Possible security concern: %s", riss.Issue.URL)
	msg := NewMessage(subject, riss.Issue.BodyText)
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("sns: failed to marshal respository issue: %v", err)
	}
	svc := sns.New(session.New())
	params := &sns.PublishInput{
		Message:  aws.String(string(b)),
		TopicArn: aws.String(s.TopicARN),
	}
	_, err = svc.Publish(params)
	return err
}
