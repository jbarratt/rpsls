package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/jbarratt/rpsls/backend/code/notify"
	"github.com/jbarratt/rpsls/backend/code/service"
	"github.com/jbarratt/rpsls/backend/code/store"
)

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

	sess := GetSession()

	st := store.New(dynamodb.New(sess), os.Getenv("TABLE_NAME"))
	no := notify.NewAPIGWNotifier(e.RequestContext.DomainName, e.RequestContext.Stage, sess)
	svc := service.NewLambdaSvc(st, no)

	switch e.RequestContext.RouteKey {
	case "$connect":
		return svc.Connect(e)
	case "$disconnect":
		return svc.Disconnect(e)
	default:
		return svc.Default(e)
	}
}

func main() {
	lambda.Start(Handler)
}

func init() {
	rand.Seed(time.Now().Unix())
}
