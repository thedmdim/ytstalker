let viewsSliderContainer = document.getElementById("views-slider");
let views = [0, 10, 50, 150, 500, 1000, 5000, 15000, "∞"]

let yearsSliderContainer = document.getElementById("years-slider");
let currentYear = new Date().getFullYear()
let years = [2006, 2010, 2014, 2016, 2019, currentYear]

let mainButton = document.getElementById("random")
let mainButtonGradient = ["d367c1", "ffc93f", "5f61de", "696be4", "e46fe8", "c8a4e7"]
let lastRight, lastLeft

var viewsSliderFormat = {
    to: function(value) {
        return views[Math.round(value)];
    },
    from: function (value) {
        return views.indexOf(Number(value));
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

function shiftColors(values, handle, unencoded, tap, positions, noUiSlider) {
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
    start: [50, 5000],
    connect: true,
    // tooltips: [true, true],
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
viewsSlider.on('update', shiftColors);


let yearsSlider = noUiSlider.create(yearsSliderContainer, {
    start: [2010, 2019],
    connect: true,
    // tooltips: [true, true],
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
yearsSlider.on('update', shiftColors);

// [lastRight, lastLeft] = viewsSlider.getPositions()

function ShowVideo(data) {

    document.getElementById("video").src = `https://www.youtube.com/embed/${data.video.id}`

    const date = new Date(data.video.uploaded * 1000)
    const day = String(date.getDate()).padStart(2, '0');
    const month = String(date.getMonth() + 1).padStart(2, '0'); // Months are zero-based
    const year = date.getFullYear();
    document.getElementById("video-info").textContent = `${day}.${month}.${year} | ${data.video.views} views`
    
    let cool = document.getElementById("cool");
    let trash = document.getElementById("trash");
    
    cool.innerText = [cool.innerText.split(" ")[0], data.reactions.cools].join(" ")
    trash.innerText = [trash.innerText.split(" ")[0], data.reactions.trashes].join(" ")

    localStorage.setItem('lastSeen', data.video.id)
}

if (localStorage.getItem('lastSeen')) {
    document.getElementById("video").src += localStorage.getItem('lastSeen')
} else {
    fetch(
        `/api/videos/random`,
        {
            headers: {"visitor": localStorage.getItem('visitor')}
        }
    )
    .then(response => response.json())
    .then(data => ShowVideo(data))
}

document.getElementById("random").onclick = function() {
    let mainButtonMessageDelay = 800
    this.style.filter = "grayscale(100%)";
    this.disabled = true;

    let viewsRange = viewsSlider.get()
    let yearsRange = yearsSlider.get()
    let horizonly = !document.getElementById("horizonly").checked
    let musiconly = !document.getElementById("musiconly").checked
    let beforeText = document.getElementById("random").innerText

    fetch(
        "/api/videos/random?" + `views=${viewsRange[0]}-${viewsRange[1]}&years=${yearsRange[0]}-${yearsRange[1]}&horizonly=${horizonly}&musiconly=${musiconly}`,
        {
            headers: {"visitor": localStorage.getItem('visitor')}
        }
    )
    .then(response => {
        console.log("response status", response.status)
        if (response.ok) {
            return response.json();
        } else {
            mainButton.innerText = data.msg
            mainButtonMessageDelay = 2000
        }
    })
    .then(data => ShowVideo(data))
    .catch(error => {
        mainButtonMessageDelay = 2000
        mainButton.innerText = "cannot fetch api"
    })
    .finally(() => {
        console.log("finally!")
        setTimeout(() => {
                this.style.filter = "";
                this.disabled = false;
                mainButton.innerText = beforeText
            }, 
            mainButtonMessageDelay
        )
    })    
}
