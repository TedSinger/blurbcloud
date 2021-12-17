class BlurbHolder {
    constructor() {
        this.text = document.querySelector("#text").innerHTML;
    };
    view() {
        return m("div", {}, m.trust(this.text));
    };
    onupdate() {
        var textDiv = document.querySelector("#text");
        var body = document.querySelector("#all")
        textDiv.style.visibility = "hidden";
        var tooSmall = 0;
        var tooBig = 128;
        while (tooBig > tooSmall + 1) {
            var middle = (tooSmall + tooBig) / 2;
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
    oncreate = this.onupdate;

    useText(t) {
        var self = this;
        self.text = t;
        m.redraw();
    };

}

function getRawText(url) {
    return m.request(url, {
        background: true,
        deserialize: function(value) {return value}, 
        responseType: "", 
        extract: function(xhr) {return xhr.responseText}
    })
};

function watchUpdates(rawurl, streamurl, method) {
    if (typeof(EventSource) !== "undefined") {
        var source = new EventSource(streamurl);
        source.onmessage = function (event) { method(event.data); }
        source.onerror = function () { setTimeout(function () { watchUpdates(rawurl, streamurl, method); }, 10 * 60 * 1000); }
    } else {
        // Long-poll?
        setTimeout(function () { getRawText(rawurl).then((x) => {method(x)}); }, 1 * 1000);
        setTimeout(function () { watchUpdates(rawurl, streamurl, method); }, 1 * 1000);
    }
};