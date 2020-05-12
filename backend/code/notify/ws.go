package notify

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
)

type APIGWNotifier struct {
	c *apigatewaymanagementapi.ApiGatewayManagementApi
}

type Notifier interface {
	Send(string, []byte) error
}

func NewAPIGWNotifier(domain, stage string, sess *session.Session) *APIGWNotifier {
	baseURL := fmt.Sprintf("https://%s/%s/", domain, stage)

	return &APIGWNotifier{
		c: apigatewaymanagementapi.New(sess, aws.NewConfig().WithEndpoint(baseURL)),
	}
}

// Send sends a message via API Gateway to the identified connection
func (n *APIGWNotifier) Send(destination string, body []byte) error {
	input := &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(destination),
		Data:         body,
	}

	_, err := n.c.PostToConnection(input)
	if err != nil {
		log.Println("Error Sending Message", err.Error())
		return err
	}
	return nil
}
