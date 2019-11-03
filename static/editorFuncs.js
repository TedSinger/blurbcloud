function initQuill(blurbText) {
    var quill = new Quill('#quill', {
        theme: 'snow',
        modules: { 'toolbar': ['bold', 'italic', 'underline', 'strike', { 'script': 'sub' }, { 'script': 'super' }, { 'color': [] }, { 'background': [] }] }
    });
    var textarea = document.querySelector('textarea');
    var toolbar = document.querySelector('.ql-toolbar');
    var button = document.querySelector('input');
    toolbar.insertBefore(button, toolbar.childNodes[0]);
    var editor = document.querySelector('.ql-editor');
    quill.clipboard.dangerouslyPasteHTML(blurbText, "silent");
    textarea.style.visibility = 'hidden';
    quill.on('text-change', function (delta) {
        textarea.value = editor.innerHTML;
    });
    var formstuff = document.querySelector("#formstuff");
    var quilldiv = document.querySelector("#quill")
    quilldiv.appendChild(toolbar);
    formstuff.appendChild(textarea);
}