import { randomUUID } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { extname, join, resolve } from 'node:path';

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

function uploadDir(): string {
  return process.env.PRIZE_UPLOAD_DIR
    ? resolve(process.env.PRIZE_UPLOAD_DIR)
    : resolve(process.cwd(), '..', '.runtime', 'prize-images');
}

function resolveExtension(file: File): string | null {
  const mimeExtension = MIME_TO_EXTENSION[file.type];
  if (mimeExtension) {
    return mimeExtension;
  }
  const fileExtension = extname(file.name).toLowerCase();
  return EXTENSION_TO_MIME[fileExtension] ? fileExtension : null;
}

function contentTypeForFilename(filename: string): string | null {
  return EXTENSION_TO_MIME[extname(filename).toLowerCase()] ?? null;
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

  const directory = uploadDir();
  const filename = `${Date.now()}-${randomUUID()}${extension}`;
  await mkdir(directory, { recursive: true });
  await writeFile(join(directory, filename), Buffer.from(await file.arrayBuffer()));
  return `/api/v1/uploads/prizes/${filename}`;
}

export async function readPrizeImage(filename: string): Promise<{ readonly buffer: Buffer; readonly contentType: string } | null> {
  if (!SAFE_FILENAME.test(filename)) {
    return null;
  }
  const contentType = contentTypeForFilename(filename);
  if (!contentType) {
    return null;
  }
  try {
    const buffer = await readFile(join(uploadDir(), filename));
    return { buffer, contentType };
  } catch {
    return null;
  }
}