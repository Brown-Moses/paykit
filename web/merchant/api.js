const API = {
  // Use relative path so it automatically works on whatever host/port it is served from
  baseUrl: '',

  async request(path, options = {}) {
    const headers = new Headers(options.headers || {});
    
    // Add API key for Merchant portal requests (not needed for admin/api since browser handles Basic Auth)
    if (!path.startsWith('/admin/') && !path.startsWith('/auth/') && path !== '/merchants') {
      const apiKey = localStorage.getItem('paykit_api_key');
      if (apiKey) {
        headers.set('Authorization', `Bearer ${apiKey}`);
      }
    }

    if (options.body && !(options.body instanceof FormData)) {
      headers.set('Content-Type', 'application/json');
      if (typeof options.body === 'object') {
        options.body = JSON.stringify(options.body);
      }
    }

    const config = {
      ...options,
      headers
    };

    const response = await fetch(`${this.baseUrl}${path}`, config);

    if (response.status === 401 && !path.startsWith('/auth/')) {
      // API Key expired or invalid, redirect to login
      localStorage.removeItem('paykit_api_key');
      window.location.href = '/portal/index.html';
      throw new Error('Unauthorized');
    }

    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.error || `HTTP error! status: ${response.status}`);
    }

    return data;
  },

  // Merchant Auth & Registration
  async login(apiKey) {
    const res = await this.request('/auth/login', {
      method: 'POST',
      body: { api_key: apiKey }
    });
    localStorage.setItem('paykit_api_key', apiKey);
    localStorage.setItem('paykit_merchant_name', res.name);
    localStorage.setItem('paykit_merchant_plan', res.plan_type);
    return res;
  },

  async register(name, webhookUrl) {
    return this.request('/merchants', {
      method: 'POST',
      body: { name, webhook_url: webhookUrl }
    });
  },

  // Transactions
  async getTransactions({ page = 1, limit = 20, status = '', externalId = '' } = {}) {
    let query = `?page=${page}&limit=${limit}`;
    if (status) query += `&status=${status}`;
    if (externalId) query += `&external_id=${externalId}`;
    return this.request(`/transactions${query}`);
  },

  async getTransaction(id) {
    return this.request(`/transactions/${id}`);
  },

  async getDeliveryLogs(txId) {
    return this.request(`/transactions/${txId}/deliveries`);
  },

  // Webhook URL update
  async updateWebhookURL(webhookUrl) {
    return this.request('/merchants/webhook-url', {
      method: 'PUT',
      body: { webhook_url: webhookUrl }
    });
  },

  // Metrics
  async getMetrics() {
    return this.request('/metrics');
  },

  // DLQ
  async getDLQ({ page = 1, limit = 20 } = {}) {
    return this.request(`/dlq?page=${page}&limit=${limit}`);
  },

  async retryDLQ(id) {
    return this.request(`/dlq/${id}/retry`, {
      method: 'POST'
    });
  },

  // Admin / Operator API
  async adminListMerchants() {
    return this.request('/admin/api/merchants');
  },

  async adminToggleMerchant(id, active) {
    return this.request(`/admin/api/merchants/${id}/toggle`, {
      method: 'PUT',
      body: { active }
    });
  },

  async adminUpdateQuota(id, planType, maxCalls) {
    return this.request(`/admin/api/merchants/${id}/quota`, {
      method: 'PUT',
      body: { plan_type: planType, max_monthly_calls: parseInt(maxCalls, 10) }
    });
  }
};
