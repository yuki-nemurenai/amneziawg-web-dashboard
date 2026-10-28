/** Keeps profile copying available on HTTP deployments without Clipboard API. */
export async function copyText(text, navigator = globalThis.navigator, document = globalThis.document) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // Browser permissions can block Clipboard API while legacy copying works.
    }
  }

  const previousFocus = document.activeElement;
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  document.body.appendChild(textarea);

  try {
    textarea.focus();
    textarea.select();
    if (!document.execCommand('copy')) {
      throw new Error('Clipboard copy was rejected');
    }
  } finally {
    textarea.remove();
    previousFocus?.focus();
  }
}
