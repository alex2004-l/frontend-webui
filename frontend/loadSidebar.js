function loadSidebar() {
    fetch('sidebar.html')
        .then(response => response.text())
        .then(data => {
            document.getElementById('sidebar-container').innerHTML = data;
            const addVmBtn = document.getElementById('add-vm');
            const popup = document.getElementById('popup');
            const closePopupBtn = document.getElementById('close-popup');

            addVmBtn.addEventListener('click', () => popup.style.display = 'flex');
            closePopupBtn.addEventListener('click', () => popup.style.display = 'none');
        });
}

window.onload = loadSidebar;