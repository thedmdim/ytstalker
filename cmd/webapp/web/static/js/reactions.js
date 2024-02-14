let cool = document.getElementById("cool");
let trash = document.getElementById("trash");


function SendCoolReaction(val) {
    fetch(`/api/videos/${localStorage.getItem('lastSeen')}/${val}`,
        {
            method: "POST",
            headers: {"visitor": localStorage.getItem('visitor')}
        }
    )
    .then(response => response.json())
    .then(data => {
        cool.innerText = [cool.innerText.split(" ")[0], data.cools].join(" ")
        trash.innerText = [trash.innerText.split(" ")[0], data.trashes].join(" ")
    })
}

cool.onclick = () => SendCoolReaction("cool")
trash.onclick = () => SendCoolReaction("trash")
