function App() {
  var _this = this;

  _this.init = () => {

    const wsURL = 'wss://foamngcalc.execute-api.us-west-2.amazonaws.com/Prod'
    _this.ws = new WebSocket(wsURL)

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
      _this.status.innerHTML = JSON.stringify(d)
    }

    _this.ws.onopen = () => {
      var url = new URL(window.location)
      if (url.hash == "") {
        console.log("hash was empty, creating a game")
        // no game is created yet
          _this.ws.send(JSON.stringify({
            'action': 'new',
          }))
      } else {
        // cut the hash off
        _this.gameId = url.hash.substring(1)
        console.log("attempting to connect to " + _this.gameId)
          _this.ws.send(JSON.stringify({
            'action': 'join',
            'gameId': _this.gameId,
          }))
      }
    }

    _this.status = document.querySelector("#status")
    _this.handleEvents();
  }

  _this.makePlay = e => {
    var elem = e.target.closest("button")
    console.log("got a play event for " + elem + " named " + elem.id)
      _this.ws.send(JSON.stringify({
        'action': 'play',
        'gameId': _this.gameId,
        'round': _this.roundId,
        'play': elem.id,
      }))
  }


  _this.handleEvents = () => {
    console.log("setting up onclick events")
    document.querySelector('#rock').onclick = _this.makePlay
    document.querySelector('#paper').onclick = _this.makePlay
    document.querySelector('#scissors').onclick = _this.makePlay
    document.querySelector('#lizard').onclick = _this.makePlay
    document.querySelector('#spock').onclick = _this.makePlay
  }


}

App.prototype.handleEvents = function() {
  this.changeButton.onclick = this.changeFont;
  this.downloadButton.onclick = this.downloadSVG;
  this.printButton.onclick = this.printSVG;
  this.textInput.onchange = this.textInput.onkeyup = this.renderCurrent;
}

var app = new App();
window.addEventListener("load", function () { app.init() }, false);
