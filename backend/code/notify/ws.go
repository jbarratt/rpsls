package notify

import (
	"fmt"
	"log"
)

type Notifier interface {
	Send(destination string, body []byte) error
}

type APIGWNotifier struct {
	c *apigatewaymanagementapi.ApiGatewayManagementApi
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
		Data:         message,
	}

	_, err := n.c.PostToConnection(input)
	if err != nil {
		log.Println("Error Sending Message", err.Error())
		return err
	}
	return nil
}
