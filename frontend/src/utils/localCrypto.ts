// Pure Web Crypto API AES-256-GCM + PBKDF2 encryption engine
// 100% offline, hardware-accelerated by iOS WebKit & all modern browsers.

export interface EncryptedPayload {
  version: number;
  salt: string;       // base64
  iv: string;         // base64
  ciphertext: string; // base64
}

function arrayBufferToBase64(buffer: ArrayBuffer | Uint8Array): string {
  const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

function base64ToArrayBuffer(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

const getCrypto = () => {
  return typeof window !== 'undefined' ? window.crypto : globalThis.crypto;
};

/**
 * Derive AES-256-GCM CryptoKey from master password using PBKDF2
 */
async function deriveKey(password: string, salt: Uint8Array): Promise<CryptoKey> {
  const enc = new TextEncoder();
  const subtle = getCrypto().subtle;
  const baseKey = await subtle.importKey(
    'raw',
    enc.encode(password),
    'PBKDF2',
    false,
    ['deriveKey']
  );

  return subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: 100000,
      hash: 'SHA-256',
    },
    baseKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

/**
 * Encrypt arbitrary JSON object using master password
 */
export async function encryptData<T>(data: T, password: string): Promise<EncryptedPayload> {
  const crypto = getCrypto();
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv = crypto.getRandomValues(new Uint8Array(12));

  const key = await deriveKey(password, salt);
  const enc = new TextEncoder();
  const plaintext = enc.encode(JSON.stringify(data));

  const ciphertext = await crypto.subtle.encrypt(
    {
      name: 'AES-GCM',
      iv: iv,
    },
    key,
    plaintext
  );

  return {
    version: 1,
    salt: arrayBufferToBase64(salt),
    iv: arrayBufferToBase64(iv),
    ciphertext: arrayBufferToBase64(ciphertext),
  };
}

/**
 * Decrypt payload using master password
 */
export async function decryptData<T>(payload: EncryptedPayload, password: string): Promise<T> {
  try {
    const crypto = getCrypto();
    const salt = base64ToArrayBuffer(payload.salt);
    const iv = base64ToArrayBuffer(payload.iv);
    const ciphertext = base64ToArrayBuffer(payload.ciphertext);

    const key = await deriveKey(password, salt);

    const decrypted = await crypto.subtle.decrypt(
      {
        name: 'AES-GCM',
        iv: iv,
      },
      key,
      ciphertext
    );

    const dec = new TextDecoder();
    return JSON.parse(dec.decode(decrypted)) as T;
  } catch (err) {
    throw new Error('主密码错误，解密失败');
  }
}
