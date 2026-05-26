import { bearerToken, corsHeaders, fail, ok } from '@/server/api-response';
import { savePrizeImage } from '@/server/prize-image-storage';
import { getService } from '@/server/singleton';

export const dynamic = 'force-dynamic';

export function OPTIONS(): Response {
  return new Response(null, {
    status: 204,
    headers: corsHeaders(),
  });
}

export async function POST(request: Request): Promise<Response> {
  try {
    const token = bearerToken(request);
    const service = await getService();
    service.adminCampaigns(token);

    const formData = await request.formData();
    const file = formData.get('file');
    if (!(file instanceof File)) {
      throw new Error('请上传礼品图片文件');
    }

    return ok('prize image uploaded', { url: await savePrizeImage(file) });
  } catch (error) {
    return fail(error);
  }
}