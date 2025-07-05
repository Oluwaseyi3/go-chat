let ws;

function login() {
    const username = document.getElementById("username-input").value.trim();
    if (!username) {
        alert("Username required");
        return;
    }
    fetch("/login", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: `username=${encodeURIComponent(username)}`
    }).then(response => {
        if (response.ok) {
            document.getElementById("login-form").style.display = "none";
            document.getElementById("chat-form").style.display = "block";
            connectWebSocket();
        } else {
            alert("Login failed");
        }
    });
}

function connectWebSocket() {
    const roomInput = document.getElementById("room-input").value.trim() || "default";
    ws = new WebSocket(`ws://localhost:8080/ws?room=${encodeURIComponent(roomInput)}`);
    ws.onopen = () => console.log("Connected to WebSocket");
    ws.onmessage = (event) => {
        const messages = document.getElementById("messages");
        const message = document.createElement("p");
        message.textContent = event.data;
        messages.appendChild(message);
        messages.scrollTop = messages.scrollHeight;
    };
    ws.onclose = () => console.log("Disconnected from WebSocket");
}

function sendMessage() {
    const input = document.getElementById("message-input");
    const message = input.value.trim();
    if (message && ws) {
        ws.send(message);
        input.value = "";
    }
}