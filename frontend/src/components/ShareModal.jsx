import React, { useState } from 'react';
import { X, Copy, Check, ExternalLink, QrCode, Share2, Smartphone } from 'lucide-react';

export const ShareModal = ({ poll, isOpen, onClose }) => {
  const [copied, setCopied] = useState(false);

  if (!isOpen || !poll) return null;

  const pollId = poll.id || poll._id;
  const shareUrl = `${window.location.origin}/poll/${pollId}`;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2500);
    } catch {
      // Fallback
      const input = document.getElementById('share-url-input');
      if (input) {
        input.select();
        document.execCommand('copy');
        setCopied(true);
        setTimeout(() => setCopied(false), 2500);
      }
    }
  };

  // Modern SVG QR code placeholder / dynamic SVG pattern
  const qrSvgUrl = `https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=${encodeURIComponent(shareUrl)}&color=3b82f6&bgcolor=111827`;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-md animate-in fade-in duration-200">
      <div className="relative w-full max-w-md bg-[#111827] border border-white/10 rounded-2xl shadow-2xl p-6 sm:p-8 overflow-hidden text-center">
        
        {/* Close Button */}
        <button
          onClick={onClose}
          className="absolute top-4 right-4 p-2 rounded-xl text-slate-400 hover:text-white hover:bg-white/5 transition-colors"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="w-12 h-12 mx-auto mb-4 rounded-2xl bg-blue-500/10 border border-blue-500/20 text-blue-400 flex items-center justify-center">
          <Share2 className="w-6 h-6" />
        </div>

        <h3 className="text-xl font-bold text-white tracking-tight">Share Live Poll</h3>
        <p className="text-xs text-slate-400 mt-1 max-w-xs mx-auto">
          Share with your audience. Results will update live as soon as votes are cast.
        </p>

        {/* QR Code Container */}
        <div className="my-6 p-4 rounded-2xl bg-white/5 border border-white/10 inline-block shadow-inner">
          <div className="w-48 h-48 bg-[#0d1322] rounded-xl flex items-center justify-center overflow-hidden border border-white/5">
            <img
              src={qrSvgUrl}
              alt="Scan to vote"
              className="w-44 h-44 rounded-lg object-contain"
              onError={(e) => {
                e.target.style.display = 'none';
                e.target.nextSibling.style.display = 'flex';
              }}
            />
            <div className="hidden w-full h-full flex-col items-center justify-center text-slate-500 p-4">
              <QrCode className="w-12 h-12 text-blue-400 mb-2" />
              <span className="text-[11px] text-center">Scan QR Code to vote instantly</span>
            </div>
          </div>
          <div className="flex items-center justify-center gap-1 text-[11px] text-slate-400 mt-2 font-medium">
            <Smartphone className="w-3.5 h-3.5 text-blue-400" />
            <span>Scan with phone camera to vote</span>
          </div>
        </div>

        {/* Share Link Copy Box */}
        <div className="flex items-center gap-2 p-1.5 rounded-xl bg-white/5 border border-white/10">
          <input
            id="share-url-input"
            type="text"
            readOnly
            value={shareUrl}
            className="flex-1 px-3 py-1.5 bg-transparent text-xs text-slate-300 font-mono focus:outline-none"
          />
          <button
            onClick={handleCopy}
            className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
              copied
                ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                : 'bg-blue-600 hover:bg-blue-500 text-white shadow-md shadow-blue-600/20'
            }`}
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5" />
                <span>Copied!</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                <span>Copy</span>
              </>
            )}
          </button>
        </div>

        {/* Quick Actions */}
        <div className="mt-4 pt-4 border-t border-white/10 flex items-center justify-between text-xs">
          <a
            href={shareUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-blue-400 hover:text-blue-300 flex items-center gap-1 font-medium hover:underline"
          >
            <ExternalLink className="w-3.5 h-3.5" />
            <span>Open Voter View</span>
          </a>

          <a
            href={`https://wa.me/?text=${encodeURIComponent(`Cast your vote on: ${poll.title} -> ${shareUrl}`)}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-emerald-400 hover:text-emerald-300 font-medium hover:underline"
          >
            WhatsApp
          </a>

          <a
            href={`https://twitter.com/intent/tweet?text=${encodeURIComponent(`Cast your vote live on "${poll.title}": ${shareUrl}`)}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-cyan-400 hover:text-cyan-300 font-medium hover:underline"
          >
            X / Twitter
          </a>
        </div>

      </div>
    </div>
  );
};
