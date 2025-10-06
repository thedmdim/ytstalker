const cool = document.getElementById("cool");
const trash = document.getElementById("trash");


function SendCoolReaction(val) {
    // kind of like: /api/videos/uPu3ocdii1/reactions/123423123123/cool
    fetch(
        `/api/videos/${localStorage.getItem('lastSeen')}/reactions/${localStorage.getItem('visitor')}/${val}`,
        { method: "POST" }
    )
    .then(response => response.json())
    .then(data => {
        cool.innerText = [cool.innerText.split(" ")[0], data.cools].join(" ")
        trash.innerText = [trash.innerText.split(" ")[0], data.trashes].join(" ")
    })
}

cool.onclick = () => SendCoolReaction("cool")
trash.onclick = () => SendCoolReaction("trash")
