import { randomUUID } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { extname, join, resolve } from 'node:path';
import { getBuiltInBannerAsset } from './builtin-banner-assets';

const MAX_PRIZE_IMAGE_BYTES = 5 * 1024 * 1024;
const SAFE_FILENAME = /^[A-Za-z0-9._-]+$/;
const EXTENSION_TO_MIME: Record<string, string> = {
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.webp': 'image/webp',
  '.gif': 'image/gif',
};
const MIME_TO_EXTENSION: Record<string, string> = {
  'image/png': '.png',
  'image/jpeg': '.jpg',
  'image/webp': '.webp',
  'image/gif': '.gif',
};

const BLOCKED_MIME_TYPES = new Set(['image/svg+xml', 'text/xml', 'application/xml']);

function uploadDir(): string {
  return process.env.PRIZE_UPLOAD_DIR
    ? resolve(process.env.PRIZE_UPLOAD_DIR)
    : resolve(process.cwd(), '..', '.runtime', 'prize-images');
}

function resolveExtension(file: File): string | null {
  if (BLOCKED_MIME_TYPES.has(file.type)) {
    return null;
  }
  const mimeExtension = MIME_TO_EXTENSION[file.type];
  if (mimeExtension) {
    return mimeExtension;
  }
  const fileExtension = extname(file.name).toLowerCase();
  if (fileExtension === '.svg') {
    return null;
  }
  return EXTENSION_TO_MIME[fileExtension] ? fileExtension : null;
}

function contentTypeForFilename(filename: string): string | null {
  const ext = extname(filename).toLowerCase();
  if (ext === '.svg') {
    return null;
  }
  return EXTENSION_TO_MIME[ext] ?? null;
}

function matchesMagicBytes(buffer: Buffer, mime: string): boolean {
  if (buffer.length < 12) {
    return false;
  }
  switch (mime) {
    case 'image/png':
      return buffer[0] === 0x89 && buffer[1] === 0x50 && buffer[2] === 0x4e && buffer[3] === 0x47;
    case 'image/jpeg':
      return buffer[0] === 0xff && buffer[1] === 0xd8 && buffer[2] === 0xff;
    case 'image/gif':
      return buffer.subarray(0, 6).toString('ascii') === 'GIF87a' || buffer.subarray(0, 6).toString('ascii') === 'GIF89a';
    case 'image/webp':
      return (
        buffer.subarray(0, 4).toString('ascii') === 'RIFF' &&
        buffer.subarray(8, 12).toString('ascii') === 'WEBP'
      );
    default:
      return false;
  }
}

function assertMagicBytes(buffer: Buffer, contentType: string): void {
  if (!matchesMagicBytes(buffer, contentType)) {
    throw new Error('图片内容与文件类型不匹配');
  }
}

export async function savePrizeImage(file: File): Promise<string> {
  const extension = resolveExtension(file);
  if (!extension) {
    throw new Error('仅支持 PNG、JPG、WEBP、GIF 图片');
  }
  if (file.size <= 0) {
    throw new Error('上传文件不能为空');
  }
  if (file.size > MAX_PRIZE_IMAGE_BYTES) {
    throw new Error('图片大小不能超过 5MB');
  }

  const buffer = Buffer.from(await file.arrayBuffer());
  const contentType = EXTENSION_TO_MIME[extension];
  if (!contentType) {
    throw new Error('仅支持 PNG、JPG、WEBP、GIF 图片');
  }
  assertMagicBytes(buffer, contentType);

  const directory = uploadDir();
  const filename = `${Date.now()}-${randomUUID()}${extension}`;
  await mkdir(directory, { recursive: true });
  await writeFile(join(directory, filename), buffer);
  return `/api/v1/uploads/prizes/${filename}`;
}

export async function readPrizeImage(filename: string): Promise<{ readonly buffer: Buffer; readonly contentType: string } | null> {
  if (!SAFE_FILENAME.test(filename)) {
    return null;
  }
  const builtInAsset = getBuiltInBannerAsset(filename);
  if (builtInAsset) {
    return builtInAsset;
  }
  const contentType = contentTypeForFilename(filename);
  if (!contentType) {
    return null;
  }
  try {
    const buffer = await readFile(join(uploadDir(), filename));
    if (!matchesMagicBytes(buffer, contentType)) {
      return null;
    }
    return { buffer, contentType };
  } catch {
    return null;
  }
}
