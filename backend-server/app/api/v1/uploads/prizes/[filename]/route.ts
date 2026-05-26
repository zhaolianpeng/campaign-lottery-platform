import { readPrizeImage } from '@/server/prize-image-storage';

type RouteContext = {
  readonly params:
    | { readonly filename: string }
    | Promise<{ readonly filename: string }>;
};

export const dynamic = 'force-dynamic';

export async function GET(_request: Request, context: RouteContext): Promise<Response> {
  const params = await context.params;
  const file = await readPrizeImage(params.filename);
  if (!file) {
    return new Response('Not Found', { status: 404 });
  }
  return new Response(new Uint8Array(file.buffer), {
    status: 200,
    headers: {
      'Content-Type': file.contentType,
      'Cache-Control': 'public, max-age=31536000, immutable',
    },
  });
}