//from: http://www.cmky.net/blog/tool/mkdocs.html
//author: humboldt
//new on : https://github.com/humboldt-xie/mkdocs-diagram
function EachTag(arr,tag,callback){
    var i=0,maxItem=arr.length;
    for (; i < maxItem; i++) {
        if(arr[i].tagName && arr[i].tagName.toLowerCase() == tag.toLowerCase()){
            callback(arr[i])
        }
    }
}

function code2Text(codeEl){
    var text = "",j;
    for (j = 0; j < codeEl.childNodes.length; j++) {
        var curNode = codeEl.childNodes[j];
        whitespace = /^\s*$/;
        if (curNode.nodeName === "#text" && !(whitespace.test(curNode.nodeValue))) {
            text = curNode.nodeValue;
            break;
        }
    }
    return text
}
function mermaidConvert(index,text,el){
    var insertSvg = function(svgCode) {
        el.innerHTML = svgCode;
    };
    mermaid.parse(text)
    mermaid.render('mermaid-' + index.toString(), text, insertSvg);
    return null
}

function flowchartConvert(index,text,el){
    diagram=flowchart.parse(text)
    diagram.drawSVG(el,{});
    return null
}
function sequenceConvert(index,text,el){
    diagram=Diagram.parse(text)
    diagram.drawSVG(el,{theme: 'simple'});
    return null
}
var diagramType={
    "gantt":mermaidConvert,
    "sequenceDiagram":mermaidConvert,
    "flowchart":mermaidConvert,
    "graph TD":mermaidConvert,
    "graph LR":mermaidConvert,
    "uml-flowchart":flowchartConvert,
    "uml-sequence-diagram":sequenceConvert
}
function convertUML() {
    var pre= document.querySelectorAll("pre")
    for (var i=0; i<pre.length; i++){
        var codeEl= pre[i];
        var parentEl = pre[i];
        EachTag(parentEl.childNodes,"code",function(code){
            codeEl=code;
        })
        var text=code2Text(codeEl);
        var type=mermaid.mermaidAPI.detectType(text);
        var conv=diagramType[type];
        if(text.indexOf("graph")==0){
            conv=mermaidConvert;
        }
        var pclass=parentEl.classList
        for(var j=0;!conv && j<parentEl.classList.length;j++){
            conv=diagramType[parentEl.classList[j]]
            if(conv){
                break;
            }
        }
        if(conv){
            el = document.createElement('div');
            el.className = type;
            try {
                parentEl.parentNode.insertBefore(el, parentEl);
                conv(i,text,el)
                parentEl.parentNode.removeChild(parentEl);
            }catch(err){
                codeEl.title=err.str;
            }
        }

    }
};

function onReady(fn) {
    if (document.addEventListener) {
        document.addEventListener('DOMContentLoaded', fn);
    } else {
        document.attachEvent('onreadystatechange', function() {
            if (document.readyState === 'interactive')
                fn();
        });
    }
}
(function (document) {
    onReady(function(){
        convertUML();
    });
})(document);