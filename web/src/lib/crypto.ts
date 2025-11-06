// 前端加密工具 - 使用 Web Crypto API

/**
 * 从密码派生加密密钥
 * 使用 PBKDF2 从共享密钥派生 AES-GCM 密钥
 */
async function deriveKey(password: string, salt: BufferSource): Promise<CryptoKey> {
  const encoder = new TextEncoder();
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    encoder.encode(password),
    'PBKDF2',
    false,
    ['deriveKey']
  );

  return crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt as BufferSource,
      iterations: 100000,
      hash: 'SHA-256'
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

/**
 * 获取或生成加密密钥
 * 使用固定的加密密钥，存储在 sessionStorage 中
 */
function getEncryptionKey(): string {
  // 优先使用 auth_token
  const authToken = localStorage.getItem('auth_token');
  if (authToken && authToken !== 'admin-mode') {
    return authToken;
  }
  
  // 如果没有 auth_token 或是管理员模式，使用会话密钥
  let sessionKey = sessionStorage.getItem('encryption_key');
  if (!sessionKey) {
    // 生成一个随机的会话密钥
    const randomBytes = crypto.getRandomValues(new Uint8Array(32));
    sessionKey = btoa(String.fromCharCode(...randomBytes));
    sessionStorage.setItem('encryption_key', sessionKey);
  }
  
  return sessionKey;
}

/**
 * 加密敏感数据
 * @param plaintext 明文数据
 * @returns Base64 编码的密文（包含 salt + iv + ciphertext）
 */
export async function encryptSensitiveData(plaintext: string): Promise<string> {
  if (!plaintext) return plaintext;

  try {
    // 获取加密密钥
    const encryptionKey = getEncryptionKey();
    
    // 生成随机 salt 和 IV
    const salt = crypto.getRandomValues(new Uint8Array(16));
    const iv = crypto.getRandomValues(new Uint8Array(12));
    
    // 派生密钥
    const key = await deriveKey(encryptionKey, salt);
    
    // 加密数据
    const encoder = new TextEncoder();
    const encryptedData = await crypto.subtle.encrypt(
      {
        name: 'AES-GCM',
        iv: iv
      },
      key,
      encoder.encode(plaintext)
    );
    
    // 组合 salt + iv + ciphertext
    const combined = new Uint8Array(salt.length + iv.length + encryptedData.byteLength);
    combined.set(salt, 0);
    combined.set(iv, salt.length);
    combined.set(new Uint8Array(encryptedData), salt.length + iv.length);
    
    // Base64 编码
    return btoa(String.fromCharCode(...combined));
  } catch (error) {
    console.error('Encryption failed:', error);
    // 如果加密失败，返回原文（降级处理）
    return plaintext;
  }
}

/**
 * 解密敏感数据（用于显示）
 * @param ciphertext Base64 编码的密文
 * @returns 明文数据
 */
export async function decryptSensitiveData(ciphertext: string): Promise<string> {
  if (!ciphertext) return ciphertext;

  try {
    // 获取加密密钥（必须与加密时使用的密钥一致）
    const encryptionKey = getEncryptionKey();
    
    // Base64 解码
    const combined = Uint8Array.from(atob(ciphertext), c => c.charCodeAt(0));
    
    // 分离 salt + iv + ciphertext
    const salt = combined.slice(0, 16);
    const iv = combined.slice(16, 28);
    const encryptedData = combined.slice(28);
    
    // 派生密钥
    const key = await deriveKey(encryptionKey, salt);
    
    // 解密数据
    const decryptedData = await crypto.subtle.decrypt(
      {
        name: 'AES-GCM',
        iv: iv
      },
      key,
      encryptedData
    );
    
    const decoder = new TextDecoder();
    return decoder.decode(decryptedData);
  } catch (error) {
    console.error('Decryption failed:', error);
    // 如果解密失败，可能是未加密的数据，直接返回
    return ciphertext;
  }
}

/**
 * 批量加密对象中的敏感字段
 */
export async function encryptSensitiveFields(
  obj: Record<string, any>,
  sensitiveFields: string[]
): Promise<Record<string, any>> {
  const result = { ...obj };
  
  for (const field of sensitiveFields) {
    if (result[field]) {
      result[field] = await encryptSensitiveData(result[field]);
    }
  }
  
  return result;
}

