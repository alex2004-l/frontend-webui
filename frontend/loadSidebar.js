function loadSidebar() {
    fetch('sidebar.html')
        .then(response => response.text())
        .then(data => {


            const url = 'http://127.0.0.1:8081/api/v1/vm/list';

            // Fetch JSON data from the URL
            fetch(url)
                .then(response => {
                    // Check if the response is OK (status code 200-299)
                    if (!response.ok) {
                        throw new Error('Network response was not ok ' + response.statusText);
                    }
                    // Parse the JSON data from the response
                    return response.json();
                })
                .then(imageList => {

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

                        const formData = new FormData();
                        formData.append('vm', fileInput.files[0]);  // Assuming fileInput is the file input element
                        formData.append('name', 'rd');

                        fetch('http://127.0.0.1:8081/api/v1/vm/build', {
                            method: 'POST',
                            body: formData
                        })
                            .then(response => response.json())
                            .then(data => console.log(data))
                            .catch(error => console.error('Error:', error));
                    });

                    console.log(imageList)
                    for (let index = 0; index < imageList.names.length; index++) {
                        addVmToSidebar(imageList.names[index])
                    }
                })
                .catch(error => {
                    // Handle any errors
                    console.error('There has been a problem with your fetch operation:', error);
                    const outputDiv = document.getElementById('output');
                    outputDiv.textContent = 'Error: ' + error.message;
                });


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

window.onload = loadSidebar;
