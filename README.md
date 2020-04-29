# Rock Paper Scissors Lizard Spock Online

Backend based on [aws example](https://github.com/aws-samples/simple-websockets-chat-app)
This [medium post](https://medium.com/@techinscribed/authenticated-serverless-websockets-using-api-gateway-golang-lambda-6e661216638) was also incredibly helpful with a worked example of APIGW websocket handling in go

## TODO

* get this into git
* get go code echoing back using the send API
* javascript hello world (send a message from node)
* FE hello world (send a message on click)
* fix dynamo part of the template (schema'y bits)
* figure out dynamo real schema
* do integration test

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


## Target user experience:

* Go to the URL. You're redirected to a URL with a game ID.
	* "New game started, link on the clipboard"
* If there are already 2 members in the game, you're noped -- (but, this game could work with N players? Hm.)
* Username is an editable field, with a default value
* Score is initially zero
* You see the star diagram and you're prompted to select your move
	* selection shows up as a halo
* Once the other player moves, you find out who won, and the score updates
	* background color behind the graphic (red or green)
	* sound effects?
* After a short timeout board goes back to "awaiting input" state

* GameID should be a 6 character alphanumeric string
* Generated client side and used on the backend for coordination


## State Payloads

* StartGame
	{ action: StartGame } --> {gameid: id, round: 1}
* UpdateGame
	{ action: UpdateGame, gameid, name, selected move {rock, paper, scissors, lizard, spock} }
* GameStatus
	{ gameid, moves {<name>: <move>}, winner: <player>, score {player: score}, round: <round> }


## Dynamo Backend

Access patterns

## Testing

One option:

https://medium.com/@basavarajkn/integration-testing-websocket-server-in-node-js-2997d107414c

- make a local javascript client
- run a node app against the prod host to test it
- use the same client in the FE app

## API Gateway integration

[AWS Docs](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-websocket-api-overview.html)
[SAM example](https://github.com/aws-samples/simple-websockets-chat-app)

* You can specify a route based on a key in the JSON payload (e.g. 'action': 'updateUser')
* There are default connect, disconnect, and default routes

You can have any of those routes go to any or all of the integrations.

To send a message to a client, you can use:
POST https://{api-id}.execute-api.us-east-1.amazonaws.com/{stage}/@connections/{connection_id}

The information you need for calling this is in the context:

```
	exports.handler = function(event, context, callback) {
	var domain = event.requestContext.domainName;
	var stage = event.requestContext.stage;
	var connectionId = event.requestContext.connectionId;
	var callbackUrl = util.format(util.format('https://%s/%s/@connections/%s', domain, stage, connectionId));
	// Do a SigV4 and then make the call
	}
```


