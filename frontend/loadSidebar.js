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

            const addVmForm = document.getElementById('form-content');
            addVmForm.addEventListener("submit", (e) => {
                e.preventDefault();
                const vmName = document.getElementById('vm-name').value;
                const vmFile = document.getElementById('vm-file').files[0];

                if (vmName && vmFile) {
                    let vms = JSON.parse(localStorage.getItem('vms')) || [];
                    vms.push(vmName);
                    localStorage.setItem('vms', JSON.stringify(vms));
                    addVmToSidebar(vmName);
                    popup.style.display = 'none';
                    addVmForm.reset();
                }
            });

            loadVmsFromLocalStorage();
        });
}

function addVmToSidebar(vmName) {
    const vmList = document.querySelector('.vm-list');
    const newVmItem = document.createElement('li');
    newVmItem.innerHTML = `<a href="#" class="vm-item">${vmName}</a>`;
    vmList.appendChild(newVmItem);

    newVmItem.addEventListener('click', () => {
        window.location.href = `vms.html?vm=${encodeURIComponent(vmName)}`;
    });
}

function loadVmsFromLocalStorage() {
    const vms = JSON.parse(localStorage.getItem('vms')) || [];
    vms.forEach(vmName => addVmToSidebar(vmName));
}

window.onload = loadSidebar;
