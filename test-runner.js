(() => {
    const page = document.location.pathname.match(/\/([^/.]+)(.html)?$/)[1];

    const onload = () =>
        setTimeout(() =>
            window.parent.postMessage({ success: false, page }), 250);
    const onviolation = (event) =>
        event.blockedURI.startsWith(`https://${page}.`) &&
        window.parent.postMessage({ success: true, page });

    window.addEventListener("load", onload);
    window.addEventListener("securitypolicyviolation", onviolation);
})()
