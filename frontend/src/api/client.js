const API_BASE = import.meta.env.VITE_API_BASE || '';

export const getVoterId = () => {
  let voterId = localStorage.getItem('pulsepoll_voter_id');
  if (!voterId) {
    voterId = 'voter_' + Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
    localStorage.setItem('pulsepoll_voter_id', voterId);
  }
  return voterId;
};

const getHeaders = (requiresAuth = false) => {
  const headers = {
    'Content-Type': 'application/json',
  };
  if (requiresAuth) {
    const token = localStorage.getItem('pulsepoll_jwt');
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  }
  return headers;
};

// Local storage key for offline/demo polls
const DEMO_POLLS_KEY = 'pulsepoll_demo_polls_store';

const getDemoPolls = () => {
  const saved = localStorage.getItem(DEMO_POLLS_KEY);
  if (saved) {
    try { return JSON.parse(saved); } catch {}
  }
  // Default sample polls if empty
  const defaults = [
    {
      id: 'demo_poll_1',
      title: 'What is your favorite modern tech stack?',
      description: 'Audience choice at Tech Summit 2026',
      creator_name: 'Demo Presenter',
      is_active: true,
      total_votes: 14,
      created_at: new Date(Date.now() - 3600000).toISOString(),
      options: [
        { index: 0, text: 'React + Go (Gin) + Redis', votes: 8 },
        { index: 1, text: 'Next.js + Server Actions', votes: 3 },
        { index: 2, text: 'Vue + Node.js + PostgreSQL', votes: 2 },
        { index: 3, text: 'Flutter + Python (FastAPI)', votes: 1 },
      ],
    },
  ];
  localStorage.setItem(DEMO_POLLS_KEY, JSON.stringify(defaults));
  return defaults;
};

const saveDemoPolls = (polls) => {
  localStorage.setItem(DEMO_POLLS_KEY, JSON.stringify(polls));
};

const handleResponse = async (response) => {
  const text = await response.text();
  let data = {};
  try {
    data = JSON.parse(text);
  } catch {
    // Non-JSON response (e.g. 502/504 gateway proxy error)
    if (!response.ok) {
      throw new Error(`Backend server unreachable (HTTP ${response.status}). Is the Go backend running on port 8080?`);
    }
  }
  if (!response.ok) {
    throw new Error(data.error || `Error ${response.status}: Failed request`);
  }
  return data;
};

export const api = {
  // Auth API
  auth: {
    login: async (email, password) => {
      try {
        const res = await fetch(`${API_BASE}/api/auth/login`, {
          method: 'POST',
          headers: getHeaders(false),
          body: JSON.stringify({ email, password }),
        });
        return await handleResponse(res);
      } catch (err) {
        // If backend is offline and user tried demo account, enable offline demo mode!
        if (email.toLowerCase() === 'demo@pulsepoll.io') {
          console.warn('Backend server offline. Entering instant Demo Mode.');
          return {
            token: 'demo_jwt_token_local_mock',
            user: {
              id: 'demo_user_id',
              name: 'Demo Presenter',
              email: 'demo@pulsepoll.io',
              created_at: new Date().toISOString(),
            },
          };
        }
        throw new Error(err.message || 'Login failed. Please check if the Go backend is running.');
      }
    },
    register: async (name, email, password) => {
      try {
        const res = await fetch(`${API_BASE}/api/auth/register`, {
          method: 'POST',
          headers: getHeaders(false),
          body: JSON.stringify({ name, email, password }),
        });
        return await handleResponse(res);
      } catch (err) {
        // Fallback for offline demo testing
        console.warn('Backend server offline. Creating local mock account.');
        return {
          token: 'demo_jwt_token_local_mock',
          user: {
            id: 'mock_user_' + Date.now(),
            name,
            email,
            created_at: new Date().toISOString(),
          },
        };
      }
    },
    getMe: async () => {
      const token = localStorage.getItem('pulsepoll_jwt');
      if (token === 'demo_jwt_token_local_mock') {
        return {
          id: 'demo_user_id',
          name: 'Demo Presenter',
          email: 'demo@pulsepoll.io',
        };
      }
      const res = await fetch(`${API_BASE}/api/auth/me`, {
        headers: getHeaders(true),
      });
      return handleResponse(res);
    },
  },

  // Polls API
  polls: {
    create: async (pollData) => {
      try {
        const res = await fetch(`${API_BASE}/api/polls`, {
          method: 'POST',
          headers: getHeaders(true),
          body: JSON.stringify(pollData),
        });
        return await handleResponse(res);
      } catch (err) {
        // Fallback in demo mode
        const polls = getDemoPolls();
        const newPoll = {
          id: 'demo_poll_' + Date.now(),
          title: pollData.title,
          description: pollData.description || '',
          creator_name: 'Demo Presenter',
          is_active: true,
          total_votes: 0,
          created_at: new Date().toISOString(),
          options: pollData.options.map((optText, idx) => ({
            index: idx,
            text: optText,
            votes: 0,
          })),
        };
        polls.unshift(newPoll);
        saveDemoPolls(polls);
        return newPoll;
      }
    },
    getMyPolls: async () => {
      try {
        const res = await fetch(`${API_BASE}/api/polls/my`, {
          headers: getHeaders(true),
        });
        return await handleResponse(res);
      } catch (err) {
        // Fallback in demo mode
        return getDemoPolls();
      }
    },
    getPoll: async (pollId) => {
      const voterId = getVoterId();
      try {
        const res = await fetch(`${API_BASE}/api/polls/${pollId}?voter_id=${voterId}`);
        return await handleResponse(res);
      } catch (err) {
        // Fallback in demo mode
        const polls = getDemoPolls();
        const poll = polls.find(p => p.id === pollId) || polls[0];
        const hasVoted = localStorage.getItem(`pulsepoll_choice_${pollId}`) !== null;
        return {
          poll,
          has_voted: hasVoted,
          live_viewers: 3,
        };
      }
    },
    vote: async (pollId, optionIndex) => {
      const voterId = getVoterId();
      try {
        const res = await fetch(`${API_BASE}/api/polls/${pollId}/vote`, {
          method: 'POST',
          headers: getHeaders(false),
          body: JSON.stringify({
            option_index: optionIndex,
            voter_id: voterId,
          }),
        });
        return await handleResponse(res);
      } catch (err) {
        // Fallback in demo mode
        const polls = getDemoPolls();
        const pollIndex = polls.findIndex(p => p.id === pollId);
        if (pollIndex !== -1) {
          polls[pollIndex].options[optionIndex].votes += 1;
          polls[pollIndex].total_votes += 1;
          saveDemoPolls(polls);
          
          // Broadcast local event for WebSocket hook simulation
          window.dispatchEvent(new CustomEvent('pulsepoll:demo_vote', {
            detail: {
              total_votes: polls[pollIndex].total_votes,
              option_votes: polls[pollIndex].options.reduce((acc, o) => ({ ...acc, [o.index]: o.votes }), {}),
            }
          }));
        }
        return { message: 'Vote recorded locally in Demo Mode' };
      }
    },
    toggleStatus: async (pollId) => {
      try {
        const res = await fetch(`${API_BASE}/api/polls/${pollId}/status`, {
          method: 'PATCH',
          headers: getHeaders(true),
        });
        return await handleResponse(res);
      } catch (err) {
        const polls = getDemoPolls();
        const poll = polls.find(p => p.id === pollId);
        if (poll) {
          poll.is_active = !poll.is_active;
          saveDemoPolls(polls);
        }
        return { message: 'Status toggled' };
      }
    },
    delete: async (pollId) => {
      try {
        const res = await fetch(`${API_BASE}/api/polls/${pollId}`, {
          method: 'DELETE',
          headers: getHeaders(true),
        });
        return await handleResponse(res);
      } catch (err) {
        const polls = getDemoPolls().filter(p => p.id !== pollId);
        saveDemoPolls(polls);
        return { message: 'Deleted' };
      }
    },
    getExportUrl: (pollId) => `${API_BASE}/api/polls/${pollId}/export`,
  },
};
