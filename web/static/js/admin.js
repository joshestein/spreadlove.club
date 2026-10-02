let credentials = null;

async function authenticatedFetch(url, options = {}) {
  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      Authorization: "Basic " + credentials,
    },
  });

  // Handle unauthorized
  if (response.status === 401) {
    throw new Error("Unauthorized");
  }

  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }

  return response;
}

document
  .getElementById("refresh")
  .addEventListener("click", loadPendingMessages);

document.getElementById("login-form").addEventListener("submit", async (e) => {
  e.preventDefault();

  const username = document.getElementById("username").value;
  const password = document.getElementById("password").value;
  const loginButton = document.getElementById("login-btn");
  const loginError = document.getElementById("login-error");

  loginButton.disabled = true;
  loginButton.textContent = "Logging in...";
  loginError.hidden = true;

  try {
    credentials = btoa(username + ":" + password);
    const response = await authenticatedFetch("/api/admin/pending");

    document.getElementById("login-form").hidden = true;
    document.getElementById("admin-app").hidden = false;

    const messages = await response.json();
    displayMessages(messages);
  } catch (error) {
    credentials = null;
    loginError.textContent =
      error.message === "Unauthorized"
        ? "Invalid username or password"
        : "Login failed. Please try again.";
    loginError.hidden = false;
    loginButton.disabled = false;
    loginButton.textContent = "Login";
  }
});

async function loadPendingMessages() {
  try {
    const response = await authenticatedFetch("/api/admin/pending");
    const messages = await response.json();
    displayMessages(messages);
  } catch (error) {
    console.error("Error loading messages:", error);
    document.getElementById("pending-list").innerHTML =
      '<div class="error">Failed to load messages</div>';
  }
}

function displayMessages(messages) {
  const container = document.getElementById("pending-list");

  if (messages.length === 0) {
    container.innerHTML = '<div class="empty">No pending messages</div>';
    return;
  }

  container.innerHTML = messages
    .map(
      (msg) => `
                <div class="message-card" id="message-${msg.id}">
                    <div class="message-content">${escapeHtml(
                      msg.content
                    )}</div>
                    <div class="message-date">
                        ${new Date(msg.created_at).toLocaleString()}
                    </div>
                    <div class="message-actions">
                        <button 
                            class="btn btn-approve"
                            onclick="approveMessage(${msg.id})">
                            ✓ Approve
                        </button>
                        <button 
                            class="btn btn-reject"
                            onclick="rejectMessage(${msg.id})">
                            ✗ Reject
                        </button>
                    </div>
                </div>
            `
    )
    .join("");
}

async function approveMessage(id) {
  try {
    await authenticatedFetch(`/api/admin/approve/${id}`, {
      method: "POST",
    });
    const element = document.getElementById(`message-${id}`);
    element.style.opacity = "0";
    setTimeout(() => element.remove(), 300);
  } catch (error) {
    console.error("Error approving message:", error);
    alert("Failed to approve message. Please try again.");
    return;
  }
}

async function rejectMessage(id) {
  try {
    await authenticatedFetch(`/api/admin/reject/${id}`, {
      method: "POST",
    });
    const element = document.getElementById(`message-${id}`);
    element.style.opacity = "0";
    setTimeout(() => element.remove(), 300);
  } catch (error) {
    console.error("Error rejecting message:", error);
    alert("Failed to approve message. Please try again.");
    return;
  }
}

function escapeHtml(unsafe) {
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
