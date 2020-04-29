package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
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
