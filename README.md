# Rock Paper Scissors Lizard Spock Online

Backend based on [aws example](https://github.com/aws-samples/simple-websockets-chat-app)
This [medium post](https://medium.com/@techinscribed/authenticated-serverless-websockets-using-api-gateway-golang-lambda-6e661216638) was also incredibly helpful with a worked example of APIGW websocket handling in go. 

The same author's frontend post is probably worth looking at too. [RxJS and Redux Observables](https://techinscribed.com/websocket-connection-reconnection-rxjs-redux-observable/?utm_source=medium&utm_medium=Referral&utm_campaign=guest_blogging)

## TODO

* make it visible when you click and add 'waiting for player'
* show a log of previous rounds
* extend player to have (uid, address)
	* use uid for key and update address on change
* See if lambda could be 128MB instead
* add limited retry to new game creation
* Add TTLs to dynamo items (https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/howitworks-ttl.html)
	* enable it on the table
	* Identify a column
	* add epoch timestamps to expirable items
* fix dynamo table name and clear old one
* Add short TTL CNAME to dynamo endpoint that's updated if it changes to make the webapp more stable
* Add google tracker to app

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

## Frontend Notes

	const url = 'wss://myserver.com/something'
	const connection = new WebSocket(url)
	connection.onmessage = e => {
	  console.log(e.data)
	}
	connection.onerror = error => {
	  console.log(`WebSocket error: ${error}`)
	}
	connection.onopen = () => {
	  connection.send('hey')
	}
