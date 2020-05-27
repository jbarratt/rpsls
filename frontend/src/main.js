function App() {
  var _this = this;

  _this.init = () => {

    _this.handleEvents();
    const wsURL = 'wss://foamngcalc.execute-api.us-west-2.amazonaws.com/Prod'
    _this.ws = new WebSocket(wsURL)

    _this.played = false;

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
         setPlayImg(_this.youPlayElem, `${d.yourPlay}.png`, true)
         setPlayImg(_this.themPlayElem, `${d.theirPlay}.png`, false)
        if(d.winner) {
          _this.youPlayElem.style.backgroundColor = "lightgreen"
          _this.themPlayElem.style.backgroundColor = "palevioletred"
        } else if(d.yourPlay != d.theirPlay) {
          _this.themPlayElem.style.backgroundColor = "lightgreen"
          _this.youPlayElem.style.backgroundColor = "palevioletred"
        }
        _this.played = false;
      }
      _this.youScoreElem.innerHTML = `You: ${d.yourScore}`
      _this.themScoreElem.innerHTML = `Them: ${d.theirScore}`
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

  _this.makePlay = e => {
    var elem = e.target.closest("button")
    if (_this.played) {
      console.log("ignoring second play attempt")
      return
    }
    console.log("got a play event for " + elem + " named " + elem.id)
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
