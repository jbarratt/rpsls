const WebSocket = require('ws');
const url = 'wss://foamngcalc.execute-api.us-west-2.amazonaws.com/Prod'

var gameId = ""
var roundId = 0



const p1 = new WebSocket(url)
p1.onmessage = e => {
	console.log("P1 Got A Message")
	d = JSON.parse(e.data)
	console.log(d)
	gameId = d.gameId
	roundId = d.round
}

p1.onerror = error => {
  console.log(`WebSocket error: ${error}`)
}

p1.onopen = () => {
  p1.send(JSON.stringify({
	  'action': 'new'
  }))
}

const p2 = new WebSocket(url)
	p2.onmessage = e => {
	console.log("P2 Got A Message")
	d = JSON.parse(e.data)
	console.log(d)
}

p2.onerror = error => {
  console.log(`WebSocket error: ${error}`)
}

setTimeout(() => {
	console.log("P2 Joining Game" + gameId)
	p2.send(JSON.stringify({
	  'gameId': gameId,
	  'action': 'join',
	}))
}, 2000);

setTimeout(() => {
	console.log("making plays")
	p1.send(JSON.stringify({
	  'gameId': gameId,
	  'action': 'play',
	  'round': roundId,
	  'play': 'spock',
	}))
	p2.send(JSON.stringify({
	  'gameId': gameId,
	  'action': 'play',
	  'round': roundId,
	  'play': 'lizard',
	}))
}, 3000);


setTimeout(() => {
	console.log("making plays")
	p1.send(JSON.stringify({
	  'gameId': gameId,
	  'action': 'play',
	  'round': roundId,
	  'play': 'lizard',
	}))
	p2.send(JSON.stringify({
	  'gameId': gameId,
	  'action': 'play',
	  'round': roundId,
	  'play': 'spock',
	}))
}, 4000);
