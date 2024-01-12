if (!localStorage.getItem('visitor')) {
    localStorage.setItem('visitor', Math.random().toString().substring(2));
}