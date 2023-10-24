let sliderContainer = document.getElementById("slider");
let views = [0, 10, 50, 150, 500, 1000, 5000, 15000, "∞"]
let randomButton = document.getElementById("random")
let randomButtonGradient = ["d367c1", "ffc93f", "5f61de", "696be4", "e46fe8", "c8a4e7"]
let lastRight, lastLeft

var format = {
    to: function(value) {
        return views[Math.round(value)];
    },
    from: function (value) {
        return views.indexOf(Number(value));
    }
};

function updateButtonText(values, handle, unencoded, tap, positions, noUiSlider) {
    randomButton.innerHTML = `Find random video within <nobr>${values[0]} - ${values[1]} views</nobr>`
}


function shiftColors(values, handle, unencoded, tap, positions, noUiSlider) {
    if (lastLeft < positions[0] || lastRight < positions[1]) {
        randomButtonGradient.unshift(randomButtonGradient[randomButtonGradient.length - 1])
        randomButtonGradient = randomButtonGradient.slice(0, randomButtonGradient.length - 1)
    } else {
        randomButtonGradient.push(randomButtonGradient[0])
        randomButtonGradient = randomButtonGradient.slice(1)
    }

    [lastLeft, lastRight] = positions

    randomButton.style.background = `linear-gradient(to right, ${randomButtonGradient.map((color) => "#"+color).join(", ")})`
}

let slider = noUiSlider.create(sliderContainer, {
    start: [50, 5000],
    connect: true,
    // // tooltips: [true, true],
    range: {
        min: 0,
        max: views.length - 1
    },
    pips: {
        mode: 'count',
        values: views.length,
        format: format
    },
    format: format,
    step: 1,
    margin: 1

});

[lastRight, lastLeft] = slider.getPositions()

slider.on('update', updateButtonText);
slider.on('update', shiftColors);

url = "https://api-l5k27grw4a-uc.a.run.app/random?"
document.getElementById("random").onclick = async function() {
    let wait = 800
    this.style.filter = "grayscale(100%)";
    this.disabled = true;

    let viewsRange = slider.get()
    let vertical = !document.querySelector("#vertical input").checked
    let beforeText = document.getElementById("random").innerText

    try {
        const res = await fetch(url + `visitor=${sessionStorage.getItem('visitor')}&from=${viewsRange[0]}&to=${viewsRange[1]}&vertical=${vertical}`)

            if (res.ok) {
                let data = await res.json()
                let date = new Date(data.uploaded)
                document.getElementById("video").src = `https://www.youtube.com/embed/${data.id}`
                document.getElementById("video-info").textContent = `${data.views} views | Uploaded ${date.getDate()}.${date.getMonth()}.${date.getFullYear()}`
                console.log("set data.id")
                localStorage.setItem('lastSeen', data.id)
                console.log("data.id is set")
                console.log(localStorage.getItem('lastSeen'))
            } else {
                let data = await res.json()
                randomButton.innerText = data.msg
                wait = 2000
            }
    } catch {
        wait = 2000
        randomButton.innerText = "cannot fetch api"
    }

    setTimeout(() => {
        this.style.filter = "";
        this.disabled = false;
        randomButton.innerText = beforeText
    }, wait)
}
