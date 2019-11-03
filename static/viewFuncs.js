function fit() {
    var textDiv = document.querySelector("#text");
    var body = document.querySelector("#all")
    textDiv.style.visibility = "hidden";
    var tooSmall = 0;
    var tooBig = 128;
    while (tooBig > tooSmall + 1) {
        middle = (tooSmall + tooBig) / 2;
        textDiv.style.fontSize = middle;
        if (window.innerHeight > body.clientHeight) {
            tooSmall = middle;
        } else {
            tooBig = middle;
        }
    }
    textDiv.style.fontSize = tooSmall;
    textDiv.style.visibility = "visible";
};

function useText(t) {
    document.getElementById("text").innerHTML = t;
    fit();
};

function loadBlurb(url) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function () {
        if (this.readyState == 4 && this.status == 200) {
            useText(this.responseText);
        }
    };
    xhttp.open("GET", url, true);
    xhttp.send();
};

function watchUpdates(rawurl, streamurl) {
    if (typeof (EventSource) !== "undefined") {
        var source = new EventSource(streamurl);
        source.onmessage = function (event) { useText(event.data); }
        source.onerror = function () { setTimeout(function () { watchUpdates(rawurl, streamurl); }, 10 * 60 * 1000); }
    } else {
        setTimeout(function () { loadBlurb(rawurl); }, 10 * 60 * 1000);
        setTimeout(function () { watchUpdates(rawurl, streamurl); }, 10 * 60 * 1000);
    }
};