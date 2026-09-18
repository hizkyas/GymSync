import React, { useState, useRef } from 'react';
import { useMutation } from '@tanstack/react-query';
import { checkIn, type CheckInResponse } from '../services/api';
import { QrCode, CheckCircle, XCircle, Loader2 } from 'lucide-react';

export const QRScanner: React.FC = () => {
  const [token, setToken] = useState('');
  const [result, setResult] = useState<CheckInResponse | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const mutation = useMutation({
    mutationFn: () => checkIn({ qr_token: token.trim() }),
    onSuccess: (data) => {
      setResult(data);
      setToken('');
      // Auto-clear result after 5 seconds
      setTimeout(() => setResult(null), 5000);
    },
    onError: (error: any) => {
      const msg = error.response?.data?.error || 'Check-in failed';
      setResult({
        access_granted: false,
        denial_reason: msg,
        member_name: 'Unknown',
        member_email: '',
        checked_in_at: new Date().toISOString(),
        check_in_id: '',
      });
      setTimeout(() => setResult(null), 5000);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!token.trim()) return;
    mutation.mutate();
  };

  return (
    <div className="qr-scanner">
      <div className="qr-scanner__icon">
        <QrCode size={28} />
      </div>
      <h3 className="qr-scanner__title">Manual Check-in</h3>
      <form onSubmit={handleSubmit} className="qr-scanner__form">
        <input
          ref={inputRef}
          id="qr-token-input"
          type="text"
          className="qr-scanner__input"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="Paste or scan QR token UUID..."
          autoFocus
          disabled={mutation.isPending}
        />
        <button
          id="qr-submit-btn"
          type="submit"
          className="qr-scanner__btn"
          disabled={!token.trim() || mutation.isPending}
        >
          {mutation.isPending ? <Loader2 size={16} className="spin" /> : 'Verify'}
        </button>
      </form>
      {result && (
        <div className={`qr-scanner__result ${result.access_granted ? 'qr-scanner__result--granted' : 'qr-scanner__result--denied'}`}>
          {result.access_granted
            ? <CheckCircle size={24} />
            : <XCircle size={24} />
          }
          <div>
            <div className="qr-scanner__result-name">{result.member_name}</div>
            <div className="qr-scanner__result-msg">
              {result.access_granted ? 'Access Granted ✓' : result.denial_reason}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
