import React, { useState, useEffect } from 'react';
import { AuthProvider } from './context/AuthContext';
import { Navbar } from './components/Navbar';
import { AuthModal } from './components/AuthModal';
import { CreatePollModal } from './components/CreatePollModal';
import { ShareModal } from './components/ShareModal';
import { Dashboard } from './pages/Dashboard';
import { PollView } from './pages/PollView';

export function AppContent() {
  const [currentView, setCurrentView] = useState('dashboard');
  const [selectedPollId, setSelectedPollId] = useState(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [sharePoll, setSharePoll] = useState(null);

  // Handle URL path on load (e.g. /poll/66ea8b3...)
  useEffect(() => {
    const handleUrlRoute = () => {
      const path = window.location.pathname;
      const match = path.match(/^\/poll\/([a-fA-F0-9]{24})/);
      if (match) {
        setSelectedPollId(match[1]);
        setCurrentView('poll-view');
      } else {
        setCurrentView('dashboard');
      }
    };

    handleUrlRoute();
    window.addEventListener('popstate', handleUrlRoute);
    return () => window.removeEventListener('popstate', handleUrlRoute);
  }, []);

  const navigateToPoll = (pollId) => {
    setSelectedPollId(pollId);
    setCurrentView('poll-view');
    window.history.pushState({}, '', `/poll/${pollId}`);
  };

  const navigateToDashboard = () => {
    setCurrentView('dashboard');
    setSelectedPollId(null);
    window.history.pushState({}, '', '/');
  };

  const handlePollCreated = (newPoll) => {
    const pollId = newPoll.id || newPoll._id;
    setSharePoll(newPoll);
    navigateToPoll(pollId);
  };

  return (
    <div className="min-h-screen flex flex-col justify-between">
      <div>
        <Navbar
          currentView={currentView}
          setCurrentView={navigateToDashboard}
          onOpenCreatePoll={() => setIsCreateOpen(true)}
        />

        <main className="pb-16">
          {currentView === 'poll-view' && selectedPollId ? (
            <PollView
              pollId={selectedPollId}
              onBack={navigateToDashboard}
              onShare={(poll) => setSharePoll(poll)}
            />
          ) : (
            <Dashboard
              onSelectPoll={navigateToPoll}
              onSharePoll={(poll) => setSharePoll(poll)}
              onOpenCreatePoll={() => setIsCreateOpen(true)}
            />
          )}
        </main>
      </div>

      {/* Global Modals */}
      <AuthModal />
      <CreatePollModal
        isOpen={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
        onPollCreated={handlePollCreated}
      />
      <ShareModal
        poll={sharePoll}
        isOpen={!!sharePoll}
        onClose={() => setSharePoll(null)}
      />

      {/* Footer */}
      <footer className="w-full border-t border-white/10 bg-[#090d16]/90 py-6 text-center text-xs text-slate-500">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-3">
          <p>© {new Date().getFullYear()} PulsePoll. All live data backed by Redis Pub/Sub & MongoDB.</p>
          <div className="flex items-center gap-4 text-slate-400">
            <span>React</span>
            <span>•</span>
            <span>Go (Gin)</span>
            <span>•</span>
            <span>Redis</span>
            <span>•</span>
            <span>MongoDB</span>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}
