import React, { useState, useEffect } from 'react';
import { api, getVoterId } from '../api/client';
import { usePollWebSocket } from '../hooks/usePollWebSocket';
import confetti from 'canvas-confetti';
import { 
  BarChart3, 
  Users, 
  Eye, 
  Share2, 
  ArrowLeft, 
  CheckCircle, 
  Lock, 
  Radio, 
  Maximize2, 
  Minimize2, 
  Award,
  Clock,
  Sparkles,
  Loader2
} from 'lucide-react';

const BAR_COLORS = [
  'from-blue-600 to-indigo-600',
  'from-purple-600 to-pink-600',
  'from-cyan-500 to-blue-500',
  'from-emerald-500 to-teal-500',
  'from-amber-500 to-orange-500',
  'from-rose-500 to-red-600',
  'from-violet-600 to-purple-600',
  'from-teal-500 to-emerald-600',
];

export const PollView = ({ pollId, onBack, onShare }) => {
  const [poll, setPoll] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedOption, setSelectedOption] = useState(null);
  const [submittingVote, setSubmittingVote] = useState(false);
  const [hasVoted, setHasVoted] = useState(false);
  const [votedOptionIndex, setVotedOptionIndex] = useState(null);
  const [isPresenterMode, setIsPresenterMode] = useState(false);

  // Initial Poll Fetch
  useEffect(() => {
    const fetchPollData = async () => {
      setLoading(true);
      try {
        const data = await api.polls.getPoll(pollId);
        setPoll(data.poll);
        setHasVoted(data.has_voted);
        
        // Check local storage if this voter voted for a specific option
        const savedChoice = localStorage.getItem(`pulsepoll_choice_${pollId}`);
        if (savedChoice !== null) {
          setVotedOptionIndex(parseInt(savedChoice, 10));
        }
      } catch (err) {
        setError(err.message || 'Failed to load poll');
      } finally {
        setLoading(false);
      }
    };

    fetchPollData();
  }, [pollId]);

  // WebSocket Live Hook
  const { connectionStatus, liveData } = usePollWebSocket(pollId);

  // Merge live WebSocket data into poll state
  const totalVotes = liveData.totalVotes !== undefined && liveData.totalVotes > 0 
    ? liveData.totalVotes 
    : (poll?.total_votes || 0);

  const isActive = liveData.isActive !== undefined ? liveData.isActive : (poll?.is_active ?? true);
  const liveViewers = liveData.viewerCount || 1;

  // Handle voting submission
  const handleVoteSubmit = async () => {
    if (selectedOption === null || submittingVote) return;
    setSubmittingVote(true);
    try {
      await api.polls.vote(pollId, selectedOption);
      
      // Save local selection
      localStorage.setItem(`pulsepoll_choice_${pollId}`, selectedOption.toString());
      setVotedOptionIndex(selectedOption);
      setHasVoted(true);

      // Trigger celebration confetti
      try {
        confetti({
          particleCount: 120,
          spread: 70,
          origin: { y: 0.6 },
          colors: ['#3b82f6', '#8b5cf6', '#ec4899', '#10b981', '#f59e0b'],
        });
      } catch {
        // ignore if canvas confetti is blocked
      }
    } catch (err) {
      alert(err.message || 'Failed to cast vote');
    } finally {
      setSubmittingVote(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center">
        <div className="w-10 h-10 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mb-4" />
        <p className="text-sm text-slate-400 font-medium">Connecting to real-time live poll...</p>
      </div>
    );
  }

  if (error || !poll) {
    return (
      <div className="max-w-xl mx-auto my-16 p-8 glass-panel rounded-2xl text-center border border-white/10">
        <div className="w-12 h-12 rounded-xl bg-red-500/10 text-red-400 flex items-center justify-center mx-auto mb-3">
          <Lock className="w-6 h-6" />
        </div>
        <h2 className="text-xl font-bold text-white mb-2">Poll Unavailable</h2>
        <p className="text-sm text-slate-400 mb-6">{error || 'This poll does not exist or has been removed.'}</p>
        <button
          onClick={onBack}
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-white/10 hover:bg-white/20 text-white text-sm font-semibold transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Return to Dashboard</span>
        </button>
      </div>
    );
  }

  // Determine top winning option
  let maxVotes = -1;
  let topOptionIndex = -1;
  poll.options.forEach((opt) => {
    const currentVotes = liveData.optionVotes?.[opt.index] !== undefined 
      ? liveData.optionVotes[opt.index] 
      : (opt.votes || 0);
    if (currentVotes > maxVotes && currentVotes > 0) {
      maxVotes = currentVotes;
      topOptionIndex = opt.index;
    }
  });

  return (
    <div className={`max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8 transition-all ${isPresenterMode ? 'max-w-5xl' : ''}`}>
      
      {/* Top Controls Bar */}
      <div className="flex items-center justify-between gap-4 mb-6">
        <button
          onClick={onBack}
          className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-white/5 hover:bg-white/10 border border-white/10 text-xs font-semibold text-slate-300 transition-colors"
        >
          <ArrowLeft className="w-3.5 h-3.5" />
          <span>Back to Dashboard</span>
        </button>

        <div className="flex items-center gap-3">
          {/* Live indicator badge */}
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-semibold">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <span className="uppercase tracking-wider text-[10px]">
              {connectionStatus === 'connected' ? 'Live Stream' : connectionStatus}
            </span>
          </div>

          {/* Viewers badge */}
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-slate-300 text-xs font-medium">
            <Eye className="w-3.5 h-3.5 text-blue-400" />
            <span>{liveViewers} watching</span>
          </div>

          {/* Presenter Mode Toggle */}
          <button
            onClick={() => setIsPresenterMode(!isPresenterMode)}
            className="p-1.5 rounded-xl bg-white/5 hover:bg-white/10 border border-white/10 text-slate-400 hover:text-white transition-colors"
            title={isPresenterMode ? 'Exit Presenter Mode' : 'Presenter / Fullscreen Mode'}
          >
            {isPresenterMode ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
          </button>

          {/* Share Button */}
          <button
            onClick={() => onShare(poll)}
            className="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-md shadow-blue-600/20 transition-all"
          >
            <Share2 className="w-3.5 h-3.5" />
            <span>Share</span>
          </button>
        </div>
      </div>

      {/* Main Poll Card */}
      <div className="glass-panel rounded-3xl p-6 sm:p-10 border border-white/10 shadow-2xl relative overflow-hidden">
        
        {/* Ambient background glow */}
        <div className="absolute top-0 right-0 w-96 h-96 bg-blue-500/10 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute bottom-0 left-0 w-96 h-96 bg-purple-500/10 rounded-full blur-3xl pointer-events-none" />

        {/* Closed Poll Banner */}
        {!isActive && (
          <div className="mb-6 p-3 rounded-2xl bg-amber-500/10 border border-amber-500/20 text-amber-400 text-xs font-semibold flex items-center justify-center gap-2">
            <Clock className="w-4 h-4" />
            <span>This poll is closed by the presenter. Live final results are shown below.</span>
          </div>
        )}

        {/* Question Header */}
        <div className="mb-8">
          <div className="flex items-center gap-2 text-xs font-medium text-slate-400 mb-2">
            <span>Created by <strong className="text-white">{poll.creator_name || 'Presenter'}</strong></span>
            <span>•</span>
            <span>{new Date(poll.created_at).toLocaleDateString()}</span>
          </div>

          <h1 className={`font-extrabold text-white tracking-tight leading-tight ${isPresenterMode ? 'text-3xl sm:text-4xl' : 'text-2xl sm:text-3xl'}`}>
            {poll.title}
          </h1>

          {poll.description && (
            <p className="text-sm text-slate-300 mt-2 leading-relaxed max-w-3xl">
              {poll.description}
            </p>
          )}
        </div>

        {/* Polling & Voting Content */}
        <div className="space-y-4">
          {poll.options.map((opt, idx) => {
            const currentOptionVotes = liveData.optionVotes?.[opt.index] !== undefined 
              ? liveData.optionVotes[opt.index] 
              : (opt.votes || 0);

            const percentage = totalVotes > 0 
              ? (currentOptionVotes / totalVotes) * 100 
              : 0;

            const isUserVotedThis = votedOptionIndex === opt.index;
            const isTopChoice = opt.index === topOptionIndex && totalVotes > 0;
            const colorGradient = BAR_COLORS[idx % BAR_COLORS.length];

            // If user has NOT voted and poll IS active -> Show interactive voting option card
            if (!hasVoted && isActive) {
              const isSelected = selectedOption === opt.index;
              return (
                <div
                  key={opt.index}
                  onClick={() => setSelectedOption(opt.index)}
                  className={`cursor-pointer p-4 sm:p-5 rounded-2xl border transition-all flex items-center justify-between group ${
                    isSelected
                      ? 'bg-blue-600/20 border-blue-500 shadow-lg shadow-blue-500/20'
                      : 'bg-white/5 hover:bg-white/10 border-white/10 hover:border-blue-500/40'
                  }`}
                >
                  <div className="flex items-center gap-3 sm:gap-4">
                    <div className={`w-6 h-6 rounded-full border flex items-center justify-center transition-all ${
                      isSelected ? 'border-blue-500 bg-blue-500 text-white' : 'border-slate-500 group-hover:border-blue-400'
                    }`}>
                      {isSelected && <div className="w-2.5 h-2.5 rounded-full bg-white" />}
                    </div>
                    <span className="text-base sm:text-lg font-medium text-white">
                      {opt.text}
                    </span>
                  </div>

                  <span className="text-xs font-mono font-bold px-2 py-1 rounded-md bg-white/5 text-slate-400">
                    Option {idx + 1}
                  </span>
                </div>
              );
            }

            // Real-Time Results Bar View (Shown when already voted, or poll is closed, or in presenter view)
            return (
              <div
                key={opt.index}
                className={`relative overflow-hidden rounded-2xl border p-4 sm:p-5 transition-all ${
                  isUserVotedThis
                    ? 'border-blue-500/60 bg-blue-950/20 shadow-lg shadow-blue-500/10'
                    : 'border-white/10 bg-white/5'
                }`}
              >
                {/* Animated fill progress bar */}
                <div
                  className={`absolute inset-0 bg-gradient-to-r ${colorGradient} opacity-20 poll-progress-bar`}
                  style={{ width: `${percentage}%` }}
                />

                {/* Content above progress bar */}
                <div className="relative z-10 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                  <div className="flex items-center gap-3">
                    <span className="text-base sm:text-lg font-semibold text-white">
                      {opt.text}
                    </span>

                    {isUserVotedThis && (
                      <span className="inline-flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded-full bg-blue-500 text-white shadow-sm">
                        <CheckCircle className="w-3 h-3" />
                        Your Vote
                      </span>
                    )}

                    {isTopChoice && (
                      <span className="inline-flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">
                        <Award className="w-3 h-3 text-amber-400" />
                        Leader
                      </span>
                    )}
                  </div>

                  <div className="flex items-center gap-4 self-end sm:self-auto">
                    <span className="text-xs font-medium text-slate-400">
                      {currentOptionVotes} {currentOptionVotes === 1 ? 'vote' : 'votes'}
                    </span>
                    <span className="text-lg sm:text-xl font-black text-white font-mono min-w-[55px] text-right">
                      {percentage.toFixed(1)}%
                    </span>
                  </div>
                </div>

              </div>
            );
          })}
        </div>

        {/* Voting Action Button */}
        {!hasVoted && isActive && (
          <div className="mt-8 pt-6 border-t border-white/10 flex flex-col sm:flex-row items-center justify-between gap-4">
            <div className="text-xs text-slate-400 flex items-center gap-1.5">
              <Sparkles className="w-4 h-4 text-blue-400" />
              <span>One vote per person. Votes update live for all participants.</span>
            </div>

            <button
              onClick={handleVoteSubmit}
              disabled={selectedOption === null || submittingVote}
              className="glow-button w-full sm:w-auto px-8 py-3.5 rounded-xl bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 text-white font-bold text-sm shadow-xl shadow-blue-500/25 flex items-center justify-center gap-2 transition-all disabled:opacity-40 disabled:pointer-events-none active:scale-95"
            >
              {submittingVote ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Submitting Vote...</span>
                </>
              ) : (
                <span>Submit Your Vote</span>
              )}
            </button>
          </div>
        )}

        {/* Footer Summary Info */}
        <div className="mt-8 pt-6 border-t border-white/10 flex items-center justify-between text-xs text-slate-400">
          <div className="flex items-center gap-2">
            <Users className="w-4 h-4 text-blue-400" />
            <span>Total Audience Votes: <strong className="text-white font-bold">{totalVotes}</strong></span>
          </div>

          <div className="flex items-center gap-3">
            <span className="hidden sm:inline">Powered by Redis + Go Gin + MongoDB</span>
            <button
              onClick={() => onShare(poll)}
              className="text-blue-400 hover:text-blue-300 font-semibold flex items-center gap-1 hover:underline"
            >
              <Share2 className="w-3.5 h-3.5" />
              <span>Invite More Voters</span>
            </button>
          </div>
        </div>

      </div>

    </div>
  );
};
