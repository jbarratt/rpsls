function App() {
  var _this = this;

  _this.init = () => {

    _this.handleEvents();
    const wsURL = 'wss://foamngcalc.execute-api.us-west-2.amazonaws.com/Prod'
    _this.ws = new WebSocket(wsURL)

    // persist a user id to use across connections
    _this.userId = localStorage.getItem("rockpaper-userid");
    if(_this.userId == null) {
      _this.userId = Math.random().toString(36).substr(2, 17);
      localStorage.setItem("rockpaper-userid", _this.userId);
    }

    _this.ws.onmessage = e => {
        d = JSON.parse(e.data)
        console.log(d)
        if (_this.gameId != d.gameId && d.gameId != "") {
          _this.gameId = d.gameId
          window.location = window.location + "#" + d.gameId
        }
        _this.roundId = d.round
        _this.updateUI(d)
    }

    _this.updateUI = d => {
      if ("roundSummary" in d) {
        _this.statusElem.innerHTML = d.roundSummary
        var li = document.createElement('li')
        li.innerHTML = d.roundSummary + ` You: ${d.yourScore} Them: ${d.theirScore}`
        _this.logElem.insertBefore(li, _this.logElem.firstChild)
      }
      _this.scoresElem.innerHTML = `You: ${d.yourScore} Them: ${d.theirScore}`
    }

    _this.ws.onopen = () => {
      var url = new URL(window.location)
      if (url.hash == "") {
        console.log("hash was empty, creating a game")
        // no game is created yet
          _this.ws.send(JSON.stringify({
            'action': 'new',
            'userId': _this.userId,
          }))
        _this.statusElem.innerHTML = "Created a game. Share the link with a friend to play!"
        var li = document.createElement('li')
        li.innerHTML = "Created a Game"
        _this.logElem.appendChild(li)
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
        var li = document.createElement('li')
        li.innerHTML = "Joined a Game"
        _this.logElem.appendChild(li)
      }
    }

  }

  _this.makePlay = e => {
    var elem = e.target.closest("button")
    console.log("got a play event for " + elem + " named " + elem.id)
      _this.ws.send(JSON.stringify({
        'action': 'play',
        'gameId': _this.gameId,
        'userId': _this.userId,
        'round': _this.roundId,
        'play': elem.id,
      }))
    _this.statusElem.innerHTML = `You played ${elem.id}, waiting on other player ....`
  }

  _this.handleEvents = () => {
    console.log("setting up onclick events")
    document.querySelector('#rock').onclick = _this.makePlay
    document.querySelector('#paper').onclick = _this.makePlay
    document.querySelector('#scissors').onclick = _this.makePlay
    document.querySelector('#lizard').onclick = _this.makePlay
    document.querySelector('#spock').onclick = _this.makePlay
    document.querySelector('#copytoclipboard').onclick = copyURLToClipboard
    _this.statusElem = document.querySelector('#status')
    _this.scoresElem = document.querySelector('#scores')
    _this.logElem = document.querySelector('#log')
  }
}

// scores h1, status p, log p

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
window.addEventListener("load", function () { app.init() }, false);
