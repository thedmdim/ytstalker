const viewsSliderContainer = document.getElementById("views-slider");
const views = ["0", "10", "50", "150", "500", "1k", "5k", "15k", "∞"]

const yearsSliderContainer = document.getElementById("years-slider");
const currentYear = new Date().getFullYear()
const years = [2006, 2010, 2014, 2016, 2019, currentYear]

const mainButton = document.getElementById("random")
let mainButtonGradient = ["d367c1", "ffc93f", "5f61de", "696be4", "e46fe8", "c8a4e7"]
let lastRight, lastLeft

var viewsSliderFormat = {
    to: function(value) {
        return views[Math.round(value)];
    },
    from: function (value) {
        return views.indexOf(value);
    }
};

var yearsSliderFormat = {
    to: function(value) {
        return years[Math.round(value)];
    },
    from: function (value) {
        return years.indexOf(Number(value));
    }
};

function shiftColors(_values, _handle, _unencoded, _tap, positions, _noUiSlider) {
    if (lastLeft < positions[0] || lastRight < positions[1]) {
        mainButtonGradient.unshift(mainButtonGradient[mainButtonGradient.length - 1])
        mainButtonGradient = mainButtonGradient.slice(0, mainButtonGradient.length - 1)
    } else {
        mainButtonGradient.push(mainButtonGradient[0])
        mainButtonGradient = mainButtonGradient.slice(1)
    }

    [lastLeft, lastRight] = positions

    mainButton.style.background = `linear-gradient(to right, ${mainButtonGradient.map((color) => "#"+color).join(", ")})`
}

let viewsSlider = noUiSlider.create(viewsSliderContainer, {
    start: ["50", "5k"],
    connect: true,
    range: {
        min: 0,
        max: views.length - 1
    },
    pips: {
        mode: 'count',
        values: views.length,
        format: viewsSliderFormat
    },
    format: viewsSliderFormat,
    step: 1,
    margin: 1
});
viewsSliderContainer.noUiSlider.on('update', shiftColors);


let yearsSlider = noUiSlider.create(yearsSliderContainer, {
    start: [2010, 2019],
    connect: true,
    range: {
        min: 0,
        max: years.length - 1
    },
    pips: {
        mode: 'count',
        values: years.length,
        format: yearsSliderFormat
    },
    format: yearsSliderFormat,
    step: 1,
    margin: 1
});
yearsSliderContainer.noUiSlider.on('update', shiftColors);


function ShowVideo(data) {
    localStorage.setItem('lastSeen', data.video.id)

    document.getElementById("video").src = `https://www.youtube.com/embed/${data.video.id}`

    history.replaceState({}, "", `/${data.video.id}`);

    const date = new Date(data.video.uploaded * 1000)
    const day = String(date.getDate()).padStart(2, '0');
    const month = String(date.getMonth() + 1).padStart(2, '0'); // Months are zero-based
    const year = date.getFullYear();
    document.getElementById("video-info").textContent = `${day}.${month}.${year} | ${data.video.views} views`

    document.getElementsByTagName("title")[0].textContent = `ytstalker | ${data.video.title} - ${year}`
    
    let cool = document.getElementById("cool");
    let trash = document.getElementById("trash");
    
    cool.innerText = [cool.innerText.split(" ")[0], data.reactions.cools].join(" ")
    trash.innerText = [trash.innerText.split(" ")[0], data.reactions.trashes].join(" ")
}

document.getElementById("random").onclick = async function() {
    let mainButtonMessageDelay = 800
    this.style.filter = "grayscale(100%)";
    this.disabled = true;

    let viewsRange = viewsSlider.get()
    let yearsRange = yearsSlider.get()
    let horizonly = document.getElementById("horizonly").checked
    let musiconly = document.getElementById("musiconly").checked
    let beforeText = document.getElementById("random").innerText

    let viewsFrom = viewsRange[0].toString()
    let viewsTo = viewsRange[1].toString()
    if (viewsFrom.includes("k")) {
        viewsFrom = viewsFrom.split("k")[0] * 1000
    }
    if (viewsTo.includes("k")) {
        viewsTo = viewsTo.split("k")[0] * 1000
    }
    if (viewsTo == "∞") {
        viewsTo = 'inf'
    }

    let query = new URLSearchParams()
    query.append("views", `${viewsFrom}-${viewsTo}`)
    query.append("years", `${yearsRange[0]}-${yearsRange[1]}`)
    query.append("visitor", localStorage.getItem('visitor'))
    if (musiconly) {
        query.append("category", 10)
    }
    if (horizonly) {
        query.append("horizonly", true)
    }

    let clearbtn = () => {
        this.style.filter = "";
        this.disabled = false;
        this.innerText = beforeText
    }

    this.innerText = "searching..."
    let response = await fetch("/api/videos/random?" + query.toString())
    if (response.ok) {
        let videoID = await response.text();
        fetch(`/api/videos/${videoID}?visitor=${localStorage.getItem('visitor')}`).then(response => response.json()).then(data => ShowVideo(data))
        clearbtn()
    } else {
        this.innerText = "cannot fetch api"
        setTimeout(clearbtn, 2000)
    }
   
}

document.getElementById("link").onclick = function() {
    navigator.clipboard.writeText(window.location.href);
    let beforeText = this.innerText
    this.innerText = "Copied!"
    setTimeout(() => {this.innerText = beforeText}, 1500)
}

document.addEventListener("DOMContentLoaded", async () => {
    let videoID = localStorage.getItem('lastSeen') || window.location.pathname.replace("/", "")
    if (!videoID) {
        videoID = await fetch(`/api/videos/random?visitor=${localStorage.getItem('visitor')}`).then(response => response.text())
    }
    fetch(`/api/videos/${videoID}?visitor=${localStorage.getItem('visitor')}`).then(response => response.json()).then(data => ShowVideo(data))
})