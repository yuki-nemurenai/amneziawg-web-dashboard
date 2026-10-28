// AmneziaWG 3.1 settings: the defaults are those of the AmneziaVPN app.

// Header protection uses S1–S4 padding as the nonce of its cipher.
export const MIN_HEADER_PROTECTION_PADDING = 12;

export const AWG3_DEFAULTS = {
  rekey_after_time: '100-120',
  rekey_timeout: '3-7',
  reject_after_time: '150-180',
  keepalive_timeout: '5-15',
  max_handshake_attempts: '15-20',
  random_trailers: 'on',
  disable_cookies: 'on',
};

// Ranges are "a-b" or a single number.
export const AWG3_RANGE_FIELDS = [
  { key: 'rekey_after_time', label: 'RekeyAfterTime (s)' },
  { key: 'rekey_timeout', label: 'RekeyTimeout (s)' },
  { key: 'reject_after_time', label: 'RejectAfterTime (s)' },
  { key: 'keepalive_timeout', label: 'KeepaliveTimeout (s)' },
  { key: 'max_handshake_attempts', label: 'MaxHandshakeAttempts' },
  { key: 'content_padding_addition', label: 'ContentPaddingAddition' },
];

export const AWG3_SWITCH_FIELDS = [
  { key: 'random_trailers', label: 'RandomTrailers' },
  { key: 'disable_cookies', label: 'DisableCookies' },
];

/**
 * Returns a random 32-byte header protection key in base64, the form of
 * awg genkey that the server accepts.
 */
export function generateHeaderProtectionKey() {
  const bytes = new Uint8Array(32);
  window.crypto.getRandomValues(bytes);
  return btoa(String.fromCharCode(...bytes));
}
