const Auth = {
  getAPIKey() {
    return localStorage.getItem('paykit_api_key');
  },

  isLoggedIn() {
    return !!this.getAPIKey();
  },

  checkAuthAndRedirect() {
    const path = window.location.pathname;
    if (path.includes('/admin/') || path.includes('/operator/')) {
      return; // Skip redirection for admin/operator pages (they are protected by HTTP Basic Auth)
    }
    const isAuthPage = path.endsWith('/index.html') || path.endsWith('/register.html') || path.endsWith('/portal') || path.endsWith('/portal/');
    
    if (this.isLoggedIn()) {
      if (isAuthPage) {
        window.location.href = '/portal/dashboard.html';
      }
    } else {
      if (!isAuthPage) {
        window.location.href = '/portal/index.html';
      }
    }
  },

  logout() {
    localStorage.removeItem('paykit_api_key');
    localStorage.removeItem('paykit_merchant_name');
    localStorage.removeItem('paykit_merchant_plan');
    window.location.href = '/portal/index.html';
  },

  // Injects sidebar/header template dynamically so we don't have to duplicate HTML markup!
  injectNavigation(activePageName) {
    const sidebarContainer = document.getElementById('sidebar-container');
    if (!sidebarContainer) return;

    const merchantName = localStorage.getItem('paykit_merchant_name') || 'Merchant';
    const merchantPlan = localStorage.getItem('paykit_merchant_plan') || 'free';

    sidebarContainer.innerHTML = `
      <div class="sidebar">
        <div class="brand">
          <div class="brand-dot"></div>
          PayKit Portal
        </div>
        <ul class="nav-links">
          <li class="${activePageName === 'dashboard' ? 'active' : ''}">
            <a href="/portal/dashboard.html">
              <span>📊</span> Dashboard
            </a>
          </li>
          <li class="${activePageName === 'transactions' ? 'active' : ''}">
            <a href="/portal/transactions.html">
              <span>💸</span> Transactions
            </a>
          </li>
          <li class="${activePageName === 'dlq' ? 'active' : ''}">
            <a href="/portal/dlq.html">
              <span>📥</span> DLQ Deliveries
            </a>
          </li>
          <li class="${activePageName === 'subscription' ? 'active' : ''}">
            <a href="/portal/subscription.html">
              <span>💳</span> Subscription
            </a>
          </li>
        </ul>
        <div class="sidebar-footer">
          <div class="user-info">
            <div class="user-avatar">${merchantName.substring(0, 2).toUpperCase()}</div>
            <div class="user-details">
              <span class="user-name">${merchantName}</span>
              <span class="user-plan">${merchantPlan} Plan</span>
            </div>
          </div>
          <button class="btn btn-secondary" onclick="Auth.logout()" style="width: 100%; justify-content: flex-start; gap: 0.5rem;">
            🚪 Log Out
          </button>
        </div>
      </div>
    `;
  },

  // Helper for displaying toast alerts
  showToast(message, type = 'success') {
    const container = document.getElementById('toast-container') || (() => {
      const el = document.createElement('div');
      el.id = 'toast-container';
      el.className = 'toast-container';
      document.body.appendChild(el);
      return el;
    })();

    const toast = document.createElement('div');
    toast.className = 'toast';
    if (type === 'error') {
      toast.style.borderColor = 'var(--accent-rose)';
      toast.innerHTML = `<span>❌</span> ${message}`;
    } else {
      toast.innerHTML = `<span>✅</span> ${message}`;
    }

    container.appendChild(toast);

    setTimeout(() => {
      toast.style.animation = 'slideIn 0.3s reverse forwards';
      setTimeout(() => toast.remove(), 300);
    }, 3000);
  },

  injectAdminNavigation(activePageName) {
    const sidebarContainer = document.getElementById('sidebar-container');
    if (!sidebarContainer) return;

    sidebarContainer.innerHTML = `
      <div class="sidebar">
        <div class="brand">
          <div class="brand-dot" style="background: var(--accent-violet); box-shadow: 0 0 12px var(--accent-violet);"></div>
          Operator Portal
        </div>
        <ul class="nav-links">
          <li class="${activePageName === 'overview' ? 'active' : ''}">
            <a href="/admin/index.html">
              <span>👥</span> All Merchants
            </a>
          </li>
          <li class="${activePageName === 'merchants' ? 'active' : ''}">
            <a href="/admin/merchants.html">
              <span>🛠️</span> Manage Quotas
            </a>
          </li>
          <li class="${activePageName === 'metrics' ? 'active' : ''}">
            <a href="/admin/metrics.html">
              <span>📈</span> System Metrics
            </a>
          </li>
        </ul>
        <div class="sidebar-footer">
          <button class="btn btn-secondary" onclick="window.location.href='/portal/index.html'" style="width: 100%; justify-content: flex-start; gap: 0.5rem;">
            🖥️ Merchant View
          </button>
        </div>
      </div>
    `;
  }
};

// Auto run auth check on load
Auth.checkAuthAndRedirect();
