'use client';

import { useEffect } from 'react';

export default function GlobalError({
  error,
  reset,
}: {
  readonly error: Error & { readonly digest?: string };
  readonly reset: () => void;
}): React.ReactNode {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <html lang="zh-CN">
      <body className="flex min-h-screen flex-col items-center justify-center bg-[#0d0f1a] px-4 text-center text-violet-50">
        <h1 className="text-xl font-black">页面发生错误</h1>
        <p className="mt-4 text-sm text-violet-100/70">请刷新页面或稍后重试</p>
        <button className="mt-6 rounded-2xl bg-violet-500 px-6 py-3 font-bold text-white" onClick={() => reset()} type="button">
          重试
        </button>
      </body>
    </html>
  );
}
