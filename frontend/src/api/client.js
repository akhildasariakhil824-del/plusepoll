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

const handleResponse = async (response) => {
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || `HTTP Error: ${response.status}`);
  }
  return data;
};

export const api = {
  // Auth API
  auth: {
    login: async (email, password) => {
      const res = await fetch(`${API_BASE}/api/auth/login`, {
        method: 'POST',
        headers: getHeaders(false),
        body: JSON.stringify({ email, password }),
      });
      return handleResponse(res);
    },
    register: async (name, email, password) => {
      const res = await fetch(`${API_BASE}/api/auth/register`, {
        method: 'POST',
        headers: getHeaders(false),
        body: JSON.stringify({ name, email, password }),
      });
      return handleResponse(res);
    },
    getMe: async () => {
      const res = await fetch(`${API_BASE}/api/auth/me`, {
        headers: getHeaders(true),
      });
      return handleResponse(res);
    },
  },

  // Polls API
  polls: {
    create: async (pollData) => {
      const res = await fetch(`${API_BASE}/api/polls`, {
        method: 'POST',
        headers: getHeaders(true),
        body: JSON.stringify(pollData),
      });
      return handleResponse(res);
    },
    getMyPolls: async () => {
      const res = await fetch(`${API_BASE}/api/polls/my`, {
        headers: getHeaders(true),
      });
      return handleResponse(res);
    },
    getPoll: async (pollId) => {
      const voterId = getVoterId();
      const res = await fetch(`${API_BASE}/api/polls/${pollId}?voter_id=${voterId}`);
      return handleResponse(res);
    },
    vote: async (pollId, optionIndex) => {
      const voterId = getVoterId();
      const res = await fetch(`${API_BASE}/api/polls/${pollId}/vote`, {
        method: 'POST',
        headers: getHeaders(false),
        body: JSON.stringify({
          option_index: optionIndex,
          voter_id: voterId,
        }),
      });
      return handleResponse(res);
    },
    toggleStatus: async (pollId) => {
      const res = await fetch(`${API_BASE}/api/polls/${pollId}/status`, {
        method: 'PATCH',
        headers: getHeaders(true),
      });
      return handleResponse(res);
    },
    delete: async (pollId) => {
      const res = await fetch(`${API_BASE}/api/polls/${pollId}`, {
        method: 'DELETE',
        headers: getHeaders(true),
      });
      return handleResponse(res);
    },
    getExportUrl: (pollId) => `${API_BASE}/api/polls/${pollId}/export`,
  },
};
