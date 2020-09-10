# Rock Paper Scissors Lizard Spock Online

A demo app for a multiplayer online browser game

- Powered by API Gateway's Websockets
- Backend in Lambda/Go
- Frontend in vanilla javascript with [PaperCSS](https://www.getpapercss.com)

Backend based on [aws example](https://github.com/aws-samples/simple-websockets-chat-app)
This [medium post](https://medium.com/@techinscribed/authenticated-serverless-websockets-using-api-gateway-golang-lambda-6e661216638) was also incredibly helpful with a worked example of APIGW websocket handling in go. 

## Rules

From [some page online](https://dodona.ugent.be/en/exercises/1647887074/)

* scissors cut paper
* paper covers rock
* rock crushes lizard
* lizard poisons Spock
* spock smashes scissors
* scissors decapitate lizard
* lizard eats paper
* paper disproves Spock
* Spock vaporizes rock
* rock crushes scissors



## API Gateway integration

[AWS Docs](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-websocket-api-overview.html)
[SAM example](https://github.com/aws-samples/simple-websockets-chat-app)

