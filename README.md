# Rock Paper Scissors Lizard Spock Online

Backend based on [aws example](https://github.com/aws-samples/simple-websockets-chat-app)
This [medium post](https://medium.com/@techinscribed/authenticated-serverless-websockets-using-api-gateway-golang-lambda-6e661216638) was also incredibly helpful with a worked example of APIGW websocket handling in go. 

The same author's frontend post is probably worth looking at too. [RxJS and Redux Observables](https://techinscribed.com/websocket-connection-reconnection-rxjs-redux-observable/?utm_source=medium&utm_medium=Referral&utm_campaign=guest_blogging)

## TODO

* create Player ID concept for the frontend to handle the idle disconnects meaning who knows who is who
	connection -> player{gameid, uid}
* javascript hello world (send a message from node)
* FE hello world (send a message on click)
* do integration test
* See if lambda could be 128MB instead
* Add google tracker to app
* Add TTLs to dynamo items (https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/howitworks-ttl.html)
	* enable it on the table
	* Identify a column
	* add epoch timestamps to expirable items

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

On connect, get a game state

Game state:
{ gameid, round, players: {you: {score: ..., played: false}, other: {score: ..., played: true, status: connected}}, lastwinner: you }

## Dynamo Backend

All entries should have 7 day TTL

Partition ID: gameid
Partition ID: session id

Partition Key / Sort Key

CONN#connectionid: get/put/deleteItem
GAME#<gameid>
 GAME#<gameid> (game state) reject update if item has chaanged

Argument: make PK and SK both structured so they can be inverted if needed
    
- Add a type field to each item
- Add data to the items for the values you also have in the PK/SK ("indexing values")
- build the idealized structs for the game layer to work with. Then write the dynamo code to implement that.

gameid, round, plays (this round), p1play, p2play, p1wins, p2wins, p1conn, p2conn

Global cache of
 - connid -> {gameid, player number}

on play:
  - do I know which player I am? cool
  	- else, fetch game state and figure it out based on conn id. Cache this data.
		- if I'm not one of the connection ids
			- if one is blank, then that's me, add my connection id there
			- add a CONN#connectionid -> gameid entry
  - update GAME#$gameid, plays:1, p1play: move UNLESS plays>0
  	- else, fetch game state and 
		- determine the winner
			- inc the winner's score
			- unset the plays
			- bump the round id
			- set plays to zero
			- store that row back in DB
		- notify all players:
			- next round id
			- did they win or lose
			- their and other player's play
			- current scores
		- on notification failure
			- unset that connection id from the game
			- remove the connectionid -> gameid entry

Access patterns:

Connection:

	no-op. Don't know at connect time if user has a game to join or not

Starting Game
	Starting a game:
	- given a session id, fetch a gameid (create a game record)

Joining Game
	Joining a game:
	- given a game id, where session not in the game, update game with session

Disconnection (or on send-to failure):

	- given a session id, find a game record, send players updated state

Play:

	- given (gameid,session) update (play, round) and also update game state
		- if other player has played, pick a winner and increment round
		- otherwise, update game state and send out state update

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
