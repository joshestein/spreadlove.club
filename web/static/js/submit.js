const $form = document.querySelector('.submit-form');
const $button = $form.querySelector('button[type="submit"]');
const $status = document.getElementById('status');

function showStatus(text, isError) {
  $status.textContent = text;
  $status.classList.toggle('error', isError);
  $status.hidden = false;
}

$form.addEventListener('submit', async (event) => {
  event.preventDefault();
  $button.disabled = true;
  $status.hidden = true;

  try {
    // The server redirects to / on success. Do not follow the redirect.
    const response = await fetch($form.action, {
      method: 'POST',
      body: new URLSearchParams(new FormData($form)),
      redirect: 'manual',
    });
    if (!response.ok && response.type !== 'opaqueredirect') throw new Error();

    $form.reset();
    showStatus('Thank you! Your message will appear after review.', false);
  } catch {
    showStatus('Could not send your message. Please try again.', true);
  } finally {
    $button.disabled = false;
  }
});
