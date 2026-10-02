const $message = document.getElementById('message');
const $refreshButton = document.getElementById('refresh');

let queue = [];
let pending = null;

function loadMore() {
  pending ??= fetch('/api/messages')
    .then((response) => (response.ok ? response.json() : []))
    .then((messages) => queue.push(...messages))
    .catch(() => {})
    .finally(() => (pending = null));
  return pending;
}

async function nextMessage() {
  if (queue.length === 0) await loadMore();
  $message.textContent = queue.shift()?.content ?? 'You are perfect as you are.';
  if (queue.length <= 2) loadMore();
}

$refreshButton.addEventListener('click', nextMessage);

nextMessage();
