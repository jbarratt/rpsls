.PHONY: deps clean build
export GO111MODULE=on
export AWS_DEFAULT_REGION=us-west-2
export AWS_REGION=us-west-2
BINARY_NAME=handler
BUCKET_NAME=serialized-sam-deploy

all: deps build
build:
	cd code && GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) main.go

deps:
	cd code && go get -u ./...

clean: 
	rm -rf ./code/handler

dynamotest:
	cd code/store/ && export TABLE_NAME=rpslp_connections && aws-vault exec serialized -- go test

package:
	aws-vault exec serialized -- sam package --template-file template.yaml --output-template-file packaged.yaml --s3-bucket $(BUCKET_NAME)

deploy:
	aws-vault exec --no-session serialized -- sam deploy --template-file packaged.yaml --stack-name rockpaper-app --capabilities CAPABILITY_IAM

logs:
	aws-vault exec serialized -- cw tail /aws/lambda/rockpaper-app-RPSLPFunction-1C3IOJ778925N -b 5m
