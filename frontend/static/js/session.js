if (!localStorage.getItem('visitor')) {
    localStorage.setItem('visitor', Math.random().toString().substring(2));
}
document.getElementById("visitor").textContent = "visitor: " + localStorage.getItem('visitor')

if (localStorage.getItem('lastSeen')) {
    document.getElementById("video").src += localStorage.getItem('lastSeen')
}