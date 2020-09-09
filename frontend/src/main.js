/**
 * Main container for the RPSLS App
 */
function App() {
  var _this = this;

  /** 
   * Initialization function. 
   *   - sets up the websocket connection
   *   - loads the userID
   *   - builds the initial UI.
   */
  _this.init = () => {

    _this.handleEvents();
    const wsURL = 'wss://foamngcalc.execute-api.us-west-2.amazonaws.com/Prod'
    _this.ws = new WebSocket(wsURL)

    _this.played = false;

    // Persist a user id to use across connections
    // The backend gets a connection ID with every request, but that changes if
    // the browser has to reconnect. This gives a stable player identifier without
    // them having to log in.
    _this.userId = localStorage.getItem("rockpaper-userid");
    if(_this.userId == null) {
      _this.userId = Math.random().toString(36).substr(2, 17);
      localStorage.setItem("rockpaper-userid", _this.userId);
    }

    // Handler which runs every time a message comes from the websocket.
    _this.ws.onmessage = e => {
        d = JSON.parse(e.data)
        // Only when needed:
        // Add the game ID to the URL so the link can be shared with others
        if (_this.gameId != d.gameId && d.gameId != "") {
          _this.gameId = d.gameId
          window.location = window.location + "#" + d.gameId
        }
        // Update the round that we're playing on now
        _this.roundId = d.round
        // Update the UI with the data from the websocket
        _this.updateUI(d)
    }

    // Handler which does UI updates
    _this.updateUI = d => {
      // If the server sends a roundSummary, that means a round is over.
      if ("roundSummary" in d) {
        _this.statusElem.innerHTML = d.roundSummary
         // Update the images for the moves
         setPlayImg(_this.youPlayElem, `${d.yourPlay}.png`, true)
         setPlayImg(_this.themPlayElem, `${d.theirPlay}.png`, false)
        // Set the colors based on who won
        if(d.winner) {
          _this.youPlayElem.style.backgroundColor = "lightgreen"
          _this.themPlayElem.style.backgroundColor = "palevioletred"
        } else if(d.yourPlay != d.theirPlay) {
          _this.themPlayElem.style.backgroundColor = "lightgreen"
          _this.youPlayElem.style.backgroundColor = "palevioletred"
        }
        // Set a flag meaning we're ready to play again
        _this.played = false;
      }
      // Update the scores no matter if the round is over or not
      _this.youScoreElem.innerHTML = `You: ${d.yourScore}`
      _this.themScoreElem.innerHTML = `Them: ${d.theirScore}`
    }

    // Handler which runs when a new websocket connection is opened
    _this.ws.onopen = () => {
      // Use the URL bar to see if the player is in a game already.
      // No hash in the URL means start a new game
      var url = new URL(window.location)
      if (url.hash == "") {
        console.log("hash was empty, creating a game")
        // no game is created yet
          _this.ws.send(JSON.stringify({
            'action': 'new',
            'userId': _this.userId,
          }))
        _this.statusElem.innerHTML = "Created a game. Share the link with a friend to play!"
      } else {
        // cut the hash off
        _this.gameId = url.hash.substring(1)
        console.log("attempting to connect to " + _this.gameId)
          _this.ws.send(JSON.stringify({
            'action': 'join',
            'userId': _this.userId,
            'gameId': _this.gameId,
          }))
          _this.statusElem.innerHTML = "Joined Game! Make a play now."
      }
    }

  }

  // Handler when a play is made
  _this.makePlay = e => {
    // Find the button element where the click event came from
    var elem = e.target.closest("button")
    // Don't let a player submit a play if the browser knows they already did
    if (_this.played) {
      console.log("ignoring second play attempt")
      return
    }
    console.log("got a play event for " + elem + " named " + elem.id)
      // Send a play event to the server
      _this.ws.send(JSON.stringify({
        'action': 'play',
        'gameId': _this.gameId,
        'userId': _this.userId,
        'round': _this.roundId,
        'play': elem.id,
      }))
    _this.statusElem.innerHTML = `You played ${elem.id}, waiting on other player ....`
    _this.played = true
    _this.themPlayElem.style.backgroundColor = ""
    _this.youPlayElem.style.backgroundColor = ""
    setPlayImg(_this.themPlayElem, "loading.gif", true)
    setPlayImg(_this.youPlayElem, `${elem.id}.png`, true)
  }

  // Set up all the click handlers for the UI buttons
  _this.handleEvents = () => {
    console.log("setting up onclick events")
    document.querySelector('#rock').onclick = _this.makePlay
    document.querySelector('#paper').onclick = _this.makePlay
    document.querySelector('#scissors').onclick = _this.makePlay
    document.querySelector('#lizard').onclick = _this.makePlay
    document.querySelector('#spock').onclick = _this.makePlay
    document.querySelector('#copytoclipboard').onclick = copyURLToClipboard
    _this.statusElem = document.querySelector('#status')
    _this.youScoreElem = document.querySelector('#youscore')
    _this.themScoreElem = document.querySelector('#themscore')
    _this.youPlayElem = document.querySelector("#youplay")
    _this.themPlayElem = document.querySelector("#themplay")
  }
}

/** Set image in the UI for the hand gesture of the player.
 *  If the reverse flag is set, display the image flipped
 *  This makes it so the same set of images can be used
 *  to show them coming from the left or right player
 */ 
const setPlayImg = (element, url, reverse) => {
  var img = document.createElement("img")
  img.src = url
  if (reverse) {
    img.style.transform = "scaleX(-1)";
  }
  if(element.firstElementChild == null) {
    element.appendChild(img)
  } else {
    element.replaceChild(img, element.firstElementChild)
  }
}

// Function discovered somewhere online
// Creates a temporary element to hold the URL text
// so it can be copied to the clipboard, then delete the element
const copyURLToClipboard = () => {
  const el = document.createElement('textarea');
  el.value = document.location;
  el.setAttribute('readonly', '');
  el.style.position = 'absolute';
  el.style.left = '-9999px';
  document.body.appendChild(el);
  el.select();
  document.execCommand('copy');
  document.body.removeChild(el);
};

var app = new App();

// Run the init message once the onload event fires
window.addEventListener("load", function () { app.init() }, false);
