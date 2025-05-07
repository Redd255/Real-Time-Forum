let socket = null;
let currentContactId = null;

// ------------------------
// WebSocket Setup
// ------------------------

function connectWebSocket() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${location.host}/ws`;

    socket = new WebSocket(wsUrl);

    socket.onopen = () => console.log("WebSocket connection established");
    socket.onmessage = handleWebSocketMessage;
    socket.onclose = () => {
        console.log("WebSocket connection closed. Reconnecting...");
        setTimeout(connectWebSocket, 3000);
    };
    socket.onerror = error => console.error("WebSocket error:", error);
}

function handleWebSocketMessage(event) {
    const message = JSON.parse(event.data);

    if (message.type === "connect") {
        console.log(message.content);
    } else if (message.type === "message") {
        const isCurrentChat =
            message.sender_id == currentContactId ||
            (message.recipient_id == currentContactId && message.is_sent);

        if (isCurrentChat) {
            appendMessage(message);
        } else if (!message.is_sent) {
            updateUnreadBadge(message.sender_id);
        }
    }
}

// ------------------------
// DOM Utilities
// ------------------------

function formatTime(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function clearUnreadBadge(contactId) {
    const badge = document.getElementById(`unread-${contactId}`);
    if (badge) {
        badge.textContent = '';
        badge.style.display = 'none';
    }
}

function updateUnreadBadge(senderId) {
    const badge = document.getElementById(`unread-${senderId}`);
    if (!badge) return;

    const count = parseInt(badge.textContent || '0') + 1;
    badge.textContent = count;
    badge.style.display = 'flex';
}

function appendMessage(msg) {
    const container = document.getElementById('chat-messages');

    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${msg.is_sent ? 'message-sent' : 'message-received'}`;

    const infoDiv = document.createElement('div');
    infoDiv.className = 'message-info';
    infoDiv.textContent = `${msg.username} • ${formatTime(msg.created_at)}`;

    const contentDiv = document.createElement('div');
    contentDiv.textContent = msg.content;

    messageDiv.appendChild(infoDiv);
    messageDiv.appendChild(contentDiv);
    container.appendChild(messageDiv);

    container.scrollTop = container.scrollHeight;
}

// ------------------------
// Event Handlers
// ------------------------

function handleContactClick(item) {
    document.querySelectorAll('.contact-item').forEach(c => c.classList.remove('active'));
    item.classList.add('active');

    const contactId = item.dataset.userid;
    const name = item.querySelector('.contact-name').textContent;
    const initials = item.querySelector('.contact-avatar').textContent;
    const bg = item.querySelector('.contact-avatar').style.background;

    document.getElementById('current-chat-name').textContent = name;
    const avatar = document.getElementById('current-chat-avatar');
    avatar.textContent = initials;
    avatar.style.background = bg;
    document.getElementById('recipient-id').value = contactId;
    document.getElementById('message-text').disabled = false;
    document.querySelector('#message-form button').disabled = false;

    clearUnreadBadge(contactId);

    currentContactId = contactId;
    loadChatHistory(contactId);
}

function handleMessageSubmit(e) {
    e.preventDefault();

    const textArea = document.getElementById('message-text');
    const message = textArea.value.trim();
    const recipientId = document.getElementById('recipient-id').value;

    if (message && recipientId && socket.readyState === WebSocket.OPEN) {
        const messageObj = {
            recipient_id: parseInt(recipientId),
            content: message,
            type: "message"
        };

        socket.send(JSON.stringify(messageObj));
        textArea.value = '';
        textArea.style.height = 'auto';
    }
}

function handleTextAreaResize() {
    this.style.height = 'auto';
    this.style.height = `${this.scrollHeight}px`;
}

// ------------------------
// Chat History & Unread
// ------------------------

function loadChatHistory(contactId) {
    fetch(`/chat-history?user_id=${contactId}`)
        .then(res => res.json())
        .then(messages => {
            const container = document.getElementById('chat-messages');
            container.innerHTML = '';

            if (messages.length === 0) {
                container.innerHTML = `<p style="text-align: center; color: #666; margin-top: 50px;">No messages yet. Start the conversation!</p>`;
                return;
            }

            messages.forEach(appendMessage);
            container.scrollTop = container.scrollHeight;
        })
        .catch(err => console.error("Error loading messages:", err));
}

function checkUnreadMessages() {
    fetch('/unread-messages')
        .then(res => res.json())
        .then(data => {
            if (data.unread_counts) {
                // Reset all
                document.querySelectorAll('.unread-badge').forEach(badge => {
                    badge.textContent = '';
                    badge.style.display = 'none';
                });

                // Update new
                data.unread_counts.forEach(({ sender_id, count }) => {
                    const badge = document.getElementById(`unread-${sender_id}`);
                    if (badge && count > 0) {
                        badge.textContent = count;
                        badge.style.display = 'flex';
                    }
                });
            }
        })
        .catch(err => console.error("Error checking unread messages:", err));
}

// ------------------------
// Initialization
// ------------------------

document.addEventListener('DOMContentLoaded', () => {
    connectWebSocket();
    checkUnreadMessages();
    setInterval(checkUnreadMessages, 10000);

    document.querySelectorAll('.contact-item').forEach(item => {
        item.addEventListener('click', () => handleContactClick(item));
    });

    const chatInput = document.getElementById('message-text');
    chatInput.addEventListener('input', handleTextAreaResize);

    document.getElementById('message-form').addEventListener('submit', handleMessageSubmit);
});
