(() => {
    const page = document.location.pathname.match(/\/([^/.]+)(.html)?$/)[1];

    const violations = [];
    window.addEventListener("securitypolicyviolation", (event) =>
        event.blockedURI.endsWith('/bad-script.js') && violations.push(event)
    );

    const timer = window.setInterval(() =>
        window.cspxLegitimateScriptWasHere && violations.length > 0 && (
            window.parent.postMessage({ success: true, page }) ||
            window.clearInterval(timer)
        )
        , 1000);

    window.cspxFailTest = () =>
        window.parent.postMessage({ success: false, page }) ||
        window.clearInterval(timer);
})()
