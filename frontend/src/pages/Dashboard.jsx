import React, { useState, useEffect } from 'react';
import { api } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { 
  BarChart2, 
  Users, 
  Eye, 
  Share2, 
  ExternalLink, 
  Power, 
  Trash2, 
  Download, 
  PlusCircle, 
  CheckCircle2, 
  Sparkles,
  Radio,
  Flame
} from 'lucide-react';

export const Dashboard = ({ onSelectPoll, onSharePoll, onOpenCreatePoll }) => {
  const { user, openAuthModal } = useAuth();
  const [polls, setPolls] = useState([]);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState({ totalPolls: 0, totalVotes: 0, totalViewers: 0 });

  const fetchPolls = async () => {
    if (!user) return;
    setLoading(true);
    try {
      const data = await api.polls.getMyPolls();
      setPolls(data);
      
      const totalV = data.reduce((acc, p) => acc + (p.total_votes || 0), 0);
      const totalView = data.reduce((acc, p) => acc + (p.live_viewers || 0), 0);
      setStats({
        totalPolls: data.length,
        totalVotes: totalV,
        totalViewers: totalView,
      });
    } catch (err) {
      console.error('Failed fetching polls:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPolls();
  }, [user]);

  const handleToggleStatus = async (pollId, e) => {
    e.stopPropagation();
    try {
      await api.polls.toggleStatus(pollId);
      fetchPolls();
    } catch (err) {
      alert('Failed to change poll status: ' + err.message);
    }
  };

  const handleDelete = async (pollId, e) => {
    e.stopPropagation();
    if (!window.confirm('Are you sure you want to delete this poll? This cannot be undone.')) return;
    try {
      await api.polls.delete(pollId);
      fetchPolls();
    } catch (err) {
      alert('Failed to delete poll: ' + err.message);
    }
  };

  const handleExport = (pollId, e) => {
    e.stopPropagation();
    window.open(api.polls.getExportUrl(pollId), '_blank');
  };

  // If user is not logged in, render Hero Landing page
  if (!user) {
    return (
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-20 text-center">
        {/* Floating badge */}
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-blue-500/10 border border-blue-500/30 text-blue-400 text-xs font-semibold mb-8 animate-pulse">
          <Sparkles className="w-3.5 h-3.5" />
          <span>Next-Generation Live Polling Stack</span>
        </div>

        <h1 className="text-4xl sm:text-6xl font-extrabold text-white tracking-tight leading-[1.15] max-w-4xl mx-auto">
          Instant Live Polling <br />
          <span className="bg-gradient-to-r from-blue-400 via-indigo-400 to-purple-400 bg-clip-text text-transparent">
            With Zero Page Refreshes.
          </span>
        </h1>

        <p className="mt-6 text-lg sm:text-xl text-slate-400 max-w-2xl mx-auto leading-relaxed">
          Powered by Go Gin, Redis pub/sub atomic counters, and MongoDB. Audiences vote in real-time with sub-millisecond live graph synchronization.
        </p>

        {/* CTAs */}
        <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
          <button
            onClick={openAuthModal}
            className="glow-button w-full sm:w-auto px-8 py-3.5 rounded-2xl bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 text-white font-bold shadow-xl shadow-blue-500/25 flex items-center justify-center gap-2 transition-all active:scale-95 text-base"
          >
            <PlusCircle className="w-5 h-5" />
            <span>Launch Your First Poll</span>
          </button>

          <button
            onClick={openAuthModal}
            className="w-full sm:w-auto px-8 py-3.5 rounded-2xl bg-white/5 hover:bg-white/10 border border-white/10 text-white font-semibold flex items-center justify-center gap-2 transition-all text-base"
          >
            <span>Sign In / Use Demo Account</span>
          </button>
        </div>

        {/* Feature Grid */}
        <div className="mt-20 grid grid-cols-1 md:grid-cols-3 gap-6 text-left">
          <div className="glass-panel p-6 rounded-2xl border border-white/10 relative overflow-hidden">
            <div className="w-12 h-12 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-400 flex items-center justify-center mb-4">
              <Radio className="w-6 h-6 animate-pulse" />
            </div>
            <h3 className="text-lg font-bold text-white mb-2">Truly Real-Time</h3>
            <p className="text-sm text-slate-400">
              Redis Pub/Sub and WebSocket streaming update every connected audience screen instantaneously as votes arrive.
            </p>
          </div>

          <div className="glass-panel p-6 rounded-2xl border border-white/10 relative overflow-hidden">
            <div className="w-12 h-12 rounded-xl bg-purple-500/10 border border-purple-500/20 text-purple-400 flex items-center justify-center mb-4">
              <Flame className="w-6 h-6" />
            </div>
            <h3 className="text-lg font-bold text-white mb-2">Atomic Redis Engine</h3>
            <p className="text-sm text-slate-400">
              Redis <code className="text-blue-400">HINCRBY</code> prevents race conditions under high concurrent voting, while <code className="text-blue-400">SADD</code> enforces duplicate vote protection.
            </p>
          </div>

          <div className="glass-panel p-6 rounded-2xl border border-white/10 relative overflow-hidden">
            <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 flex items-center justify-center mb-4">
              <CheckCircle2 className="w-6 h-6" />
            </div>
            <h3 className="text-lg font-bold text-white mb-2">Durable MongoDB Storage</h3>
            <p className="text-sm text-slate-400">
              Poll schemas, user credentials, and detailed voter audit logs are safely persisted and exportable to CSV/JSON.
            </p>
          </div>
        </div>
      </div>
    );
  }

  // Authenticated Creator Dashboard View
  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      
      {/* Top Banner with Stats */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6 pb-8 border-b border-white/10">
        <div>
          <h1 className="text-3xl font-extrabold text-white tracking-tight">Creator Dashboard</h1>
          <p className="text-sm text-slate-400 mt-1">
            Welcome back, <span className="text-blue-400 font-semibold">{user.name}</span>. Manage your live interactive polls.
          </p>
        </div>

        <button
          onClick={onOpenCreatePoll}
          className="glow-button inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-sm font-bold shadow-lg shadow-blue-500/20 transition-all self-start md:self-auto"
        >
          <PlusCircle className="w-4 h-4" />
          <span>Create New Poll</span>
        </button>
      </div>

      {/* Metric Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-5 my-8">
        <div className="glass-panel p-5 rounded-2xl border border-white/10 flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-400 flex items-center justify-center">
            <BarChart2 className="w-6 h-6" />
          </div>
          <div>
            <div className="text-2xl font-black text-white">{stats.totalPolls}</div>
            <div className="text-xs text-slate-400 font-medium">Total Polls Created</div>
          </div>
        </div>

        <div className="glass-panel p-5 rounded-2xl border border-white/10 flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-purple-500/10 border border-purple-500/20 text-purple-400 flex items-center justify-center">
            <Users className="w-6 h-6" />
          </div>
          <div>
            <div className="text-2xl font-black text-white">{stats.totalVotes}</div>
            <div className="text-xs text-slate-400 font-medium">Total Votes Cast</div>
          </div>
        </div>

        <div className="glass-panel p-5 rounded-2xl border border-white/10 flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 flex items-center justify-center">
            <Eye className="w-6 h-6" />
          </div>
          <div>
            <div className="text-2xl font-black text-white flex items-center gap-2">
              <span>{stats.totalViewers}</span>
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-ping" />
            </div>
            <div className="text-xs text-slate-400 font-medium">Active Live Viewers</div>
          </div>
        </div>
      </div>

      {/* Polls List Section */}
      <div className="mt-8">
        <h2 className="text-xl font-bold text-white mb-4">Your Live Polls</h2>

        {loading ? (
          <div className="p-12 text-center text-slate-400">
            <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            <span>Loading polls from database...</span>
          </div>
        ) : polls.length === 0 ? (
          <div className="glass-panel p-12 rounded-2xl text-center border border-white/10">
            <BarChart2 className="w-12 h-12 text-slate-600 mx-auto mb-3" />
            <h3 className="text-lg font-bold text-white">No polls created yet</h3>
            <p className="text-sm text-slate-400 mt-1 max-w-sm mx-auto">
              Create your first poll now to generate a live link and start collecting votes from your audience.
            </p>
            <button
              onClick={onOpenCreatePoll}
              className="mt-5 inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-sm font-semibold transition-colors"
            >
              <PlusCircle className="w-4 h-4" />
              <span>Create First Poll</span>
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {polls.map((poll) => {
              const pollId = poll.id || poll._id;
              return (
                <div
                  key={pollId}
                  onClick={() => onSelectPoll(pollId)}
                  className="glass-panel-interactive p-6 rounded-2xl border border-white/10 cursor-pointer flex flex-col justify-between"
                >
                  <div>
                    {/* Header with status badges */}
                    <div className="flex items-center justify-between gap-3 mb-3">
                      <div className="flex items-center gap-2">
                        {poll.is_active ? (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                            Active
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-slate-800 text-slate-400 border border-slate-700">
                            Closed
                          </span>
                        )}
                        <span className="text-xs text-slate-500">
                          {new Date(poll.created_at).toLocaleDateString()}
                        </span>
                      </div>

                      <div className="flex items-center gap-2 text-xs font-semibold text-slate-400">
                        <span className="flex items-center gap-1">
                          <Users className="w-3.5 h-3.5 text-blue-400" />
                          {poll.total_votes || 0} votes
                        </span>
                        {poll.live_viewers > 0 && (
                          <span className="flex items-center gap-1 text-emerald-400">
                            <Eye className="w-3.5 h-3.5" />
                            {poll.live_viewers} live
                          </span>
                        )}
                      </div>
                    </div>

                    {/* Poll Title */}
                    <h3 className="text-lg font-bold text-white group-hover:text-blue-400 transition-colors line-clamp-2">
                      {poll.title}
                    </h3>

                    {poll.description && (
                      <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                        {poll.description}
                      </p>
                    )}

                    {/* Options Preview */}
                    <div className="mt-4 space-y-1.5">
                      {poll.options.slice(0, 3).map((opt, i) => {
                        const pct = poll.total_votes > 0 ? ((opt.votes || 0) / poll.total_votes) * 100 : 0;
                        return (
                          <div key={i} className="text-xs text-slate-300 flex items-center justify-between">
                            <span className="truncate max-w-[200px]">{opt.text}</span>
                            <span className="text-slate-400 font-mono font-medium">{pct.toFixed(0)}%</span>
                          </div>
                        );
                      })}
                      {poll.options.length > 3 && (
                        <div className="text-[11px] text-slate-500 italic">
                          +{poll.options.length - 3} more choices
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Actions Bar */}
                  <div className="mt-6 pt-4 border-t border-white/10 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onSharePoll(poll);
                        }}
                        className="p-2 rounded-xl bg-white/5 hover:bg-blue-500/20 text-slate-400 hover:text-blue-400 border border-white/5 transition-colors"
                        title="Share Poll Link & QR Code"
                      >
                        <Share2 className="w-4 h-4" />
                      </button>

                      <button
                        onClick={(e) => handleToggleStatus(pollId, e)}
                        className={`p-2 rounded-xl border border-white/5 transition-colors ${
                          poll.is_active
                            ? 'bg-white/5 hover:bg-amber-500/20 text-slate-400 hover:text-amber-400'
                            : 'bg-white/5 hover:bg-emerald-500/20 text-slate-400 hover:text-emerald-400'
                        }`}
                        title={poll.is_active ? 'Close Poll' : 'Re-open Poll'}
                      >
                        <Power className="w-4 h-4" />
                      </button>

                      <button
                        onClick={(e) => handleExport(pollId, e)}
                        className="p-2 rounded-xl bg-white/5 hover:bg-purple-500/20 text-slate-400 hover:text-purple-400 border border-white/5 transition-colors"
                        title="Export Votes as CSV"
                      >
                        <Download className="w-4 h-4" />
                      </button>

                      <button
                        onClick={(e) => handleDelete(pollId, e)}
                        className="p-2 rounded-xl bg-white/5 hover:bg-red-500/20 text-slate-400 hover:text-red-400 border border-white/5 transition-colors"
                        title="Delete Poll"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>

                    <span className="text-xs font-semibold text-blue-400 flex items-center gap-1">
                      <span>Open Live View</span>
                      <ExternalLink className="w-3.5 h-3.5" />
                    </span>
                  </div>

                </div>
              );
            })}
          </div>
        )}
      </div>

    </div>
  );
};
