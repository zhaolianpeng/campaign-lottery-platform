'use client';

import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { fulfillPaymentOrder, pollPaymentUntilPaid } from '@/client/payment-api';
import { resumePendingPayment } from '@/client/payment-checkout';

export default function PayOrderPage(): React.ReactNode {
  const params = useParams<{ orderNo: string }>();
  const router = useRouter();
  const orderNo = params.orderNo;
  const token = typeof window !== 'undefined' ? (window.sessionStorage.getItem('campaign-lottery-token') ?? '') : '';
  const invalidAccess = !token || !orderNo;
  const [status, setStatus] = useState<'pending' | 'paid' | 'error'>(invalidAccess ? 'error' : 'pending');
  const [message, setMessage] = useState(invalidAccess ? '请先登录后再查看支付结果' : '正在确认支付结果…');

  useEffect(() => {
    if (invalidAccess) {
      return;
    }
    void (async () => {
      try {
        const resumed = await resumePendingPayment(token);
        if (resumed) {
          setStatus('paid');
          setMessage('支付成功，权益已到账');
          return;
        }
        await pollPaymentUntilPaid(token, orderNo, { maxAttempts: 30 });
        await fulfillPaymentOrder(token, orderNo);
        setStatus('paid');
        setMessage('支付成功，权益已到账');
      } catch (error) {
        setStatus('error');
        setMessage(error instanceof Error ? error.message : '支付确认失败');
      }
    })();
  }, [invalidAccess, orderNo, token]);

  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-[#0d0f1a] px-4 text-center text-violet-50">
      <h1 className="text-xl font-black">支付订单 {orderNo}</h1>
      <p className={`mt-4 text-sm ${status === 'paid' ? 'text-emerald-300' : status === 'error' ? 'text-red-300' : 'text-violet-100/70'}`}>{message}</p>
      <button className="mt-6 rounded-2xl bg-violet-500 px-6 py-3 font-bold text-white" onClick={() => router.push('/')} type="button">
        返回首页
      </button>
    </main>
  );
}
