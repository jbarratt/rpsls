package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
)

func Connect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$connect: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func Disconnect(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$disconnect: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func Default(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("$defaut: body: '%s' connectionId: '%s'\n", e.Body, e.RequestContext.ConnectionID)

	baseURL := fmt.Sprintf("https://%s/%s/", e.RequestContext.DomainName, e.RequestContext.Stage)

	// send a message back to the connectionID
	// could return it inline but this is an example of sending to others
	sess := GetSession()

	input := &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: &e.RequestContext.ConnectionID,
		Data:         []byte(e.Body),
	}

	apigateway := apigatewaymanagementapi.New(sess, aws.NewConfig().WithEndpoint(baseURL))

	_, err := apigateway.PostToConnection(input)
	if err != nil {
		log.Println("Error Posting", err.Error())
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func GetSession() *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Fatalln("unable to create session", err.Error())
	}
	return sess
}

func Handler(e events.APIGatewayWebsocketProxyRequest) (interface{}, error) {
	fmt.Printf("Entered handler\n")
	switch e.RequestContext.RouteKey {
	case "$connect":
		return Connect(e)
	case "$disconnect":
		return Disconnect(e)
	default:
		return Default(e)
	}
}

func main() {
	lambda.Start(Handler)
}
