function fitText() {
    // resize #blurb to the maximum without overflowing the screen
    var textDiv = document.querySelector(".ql-editor");
    var body = document.querySelector("#all")
    textDiv.style.visibility = "hidden";
    var tooSmall = 0;
    var tooBig = 128;
    while (tooBig > tooSmall + 1) {
        var middle = (tooSmall + tooBig) / 2;
        textDiv.style.fontSize = middle + 'px';
        if (window.innerHeight > body.clientHeight) {
            tooSmall = middle;
        } else {
            tooBig = middle;
        }
    }
    textDiv.style.fontSize = tooSmall + 'px';
    textDiv.style.visibility = "visible";
}

class BlurbHolder {
    constructor(url, blurb_version) {
        this.blurb_version = blurb_version
        this.url = url
        this.quill = new Quill('#blurb', {
            theme: 'snow',
            modules: { 'toolbar': [
                ['bold', 'italic', 'underline'],
                ['strike', { 'script': 'sub' }, { 'script': 'super' }],
                ['clean', { 'color': [] }, { 'background': [] }]
            ]}
        });
        var self = this;
        var editor = document.querySelector('.ql-editor');
        this.quill.on('text-change', function (delta) {
            self.blurb_version += 1
            const body = JSON.stringify({"blurb_text": editor.innerHTML, blurb_version: self.blurb_version});
            const headers = new Headers({"Content-Type": "application/json"});
            const request = new Request(self.url, {method: 'PUT'});
            const promise = fetch(request, {
                body: body,
                headers: headers
            });
            fitText();
        });
        fitText()
    };

    useText = function(json) {
        const data = JSON.parse(json);
        console.log(data)
        const blurbText = data["blurb_text"]
        const version = data["blurb_version"]
        if( version > this.blurb_version) {
            this.blurb_version = version
            const sel = this.quill.getSelection();
            this.quill.clipboard.dangerouslyPasteHTML(blurbText, "silent");
            this.quill.setSelection(sel);
            fitText();
        };
    };
};


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
        source.onerror = function () { console.log("disconnected!"); }
    } else {
        // poll
        setTimeout(function () { getRawText(rawurl).then((x) => {method(x)}); }, 1 * 1000);
        setTimeout(function () { watchUpdates(rawurl, streamurl, method); }, 1 * 1000);
    }
};