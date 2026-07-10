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
              <i class="ti ti-layout-dashboard"></i> Dashboard
            </a>
          </li>
          <li class="${activePageName === 'transactions' ? 'active' : ''}">
            <a href="/portal/transactions.html">
              <i class="ti ti-arrows-exchange"></i> Transactions
            </a>
          </li>
          <li class="${activePageName === 'dlq' ? 'active' : ''}">
            <a href="/portal/dlq.html">
              <i class="ti ti-inbox"></i> DLQ
            </a>
          </li>
          <li class="${activePageName === 'subscription' ? 'active' : ''}">
            <a href="/portal/subscription.html">
              <i class="ti ti-credit-card"></i> Subscription
            </a>
          </li>
        </ul>
        <div class="sidebar-footer">
          <div class="user-info">
            <div class="user-avatar">${merchantName.substring(0, 2).toUpperCase()}</div>
            <div class="user-details">
              <span class="user-name">${merchantName}</span>
              <span class="user-plan">${merchantPlan} plan</span>
            </div>
          </div>
          <button class="btn" onclick="Auth.logout()" style="width: 100%; justify-content: flex-start;">
            <i class="ti ti-logout"></i> Log out
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
      toast.innerHTML = `<span class="badge badge-danger">error</span> ${message}`;
    } else {
      toast.innerHTML = `<span class="badge badge-success">ok</span> ${message}`;
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
          <div class="brand-dot"></div>
          Operator Portal
        </div>
        <ul class="nav-links">
          <li class="${activePageName === 'overview' ? 'active' : ''}">
            <a href="/admin/index.html">
              <i class="ti ti-users"></i> All merchants
            </a>
          </li>
          <li class="${activePageName === 'merchants' ? 'active' : ''}">
            <a href="/admin/merchants.html">
              <i class="ti ti-adjustments"></i> Manage quotas
            </a>
          </li>
          <li class="${activePageName === 'metrics' ? 'active' : ''}">
            <a href="/admin/metrics.html">
              <i class="ti ti-chart-bar"></i> System metrics
            </a>
          </li>
        </ul>
        <div class="sidebar-footer">
          <button class="btn" onclick="window.location.href='/portal/index.html'" style="width: 100%; justify-content: flex-start;">
            <i class="ti ti-building-store"></i> Merchant view
          </button>
        </div>
      </div>
    `;
  }
};

// Auto run auth check on load
Auth.checkAuthAndRedirect();
