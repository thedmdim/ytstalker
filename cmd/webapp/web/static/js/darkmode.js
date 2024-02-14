if (sessionStorage.getItem('dark-mode') === "true") {
    document.querySelector("#dark-mode").checked = true;
    darkMode()
  }
  
function darkMode() {
  sessionStorage.setItem('dark-mode', document.querySelector("#dark-mode").checked);
  document.body.classList.toggle("dark");
  document.querySelector('html').classList.toggle("night");
}