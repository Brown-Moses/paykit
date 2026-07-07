# PayKit Portal UI Guide & Architecture

This guide documents the design system, page flow, state management, and user experience paradigms of the redesigned PayKit Merchant and Operator Portals.

---

## 🎨 Global Design System & CSS Variables

The front-end design is fully configured within `/web/merchant/paykit.css`, using custom CSS properties to deliver a premium, responsive glassmorphic dark theme.

### Color Palettes & Accent Tokens
```css
:root {
  --bg-main: #030712;
  --bg-card: rgba(17, 24, 39, 0.7);
  --bg-card-solid: #111827;
  --bg-input: rgba(31, 41, 55, 0.5);
  --border-color: rgba(255, 255, 255, 0.08);
  --border-hover: rgba(6, 182, 212, 0.3);
  --text-main: #f9fafb;
  --text-muted: #9ca3af;
  
  /* Accent Colors */
  --accent-cyan: #06b6d4;     /* Core Merchant Accent */
  --accent-violet: #6366f1;   /* Primary Actions / Brand Glow */
  --accent-emerald: #10b981;  /* Success / Active states */
  --accent-rose: #f43f5e;     /* Error / Suspended / Quota Overlimit */
  --accent-amber: #f59e0b;    /* Warnings / Operator Accent */
}
```

### Visual Styling Highlights
- **Backdrop Filters**: Cards feature a high-performance `backdrop-filter: blur(12px)` combined with transparent borders for depth.
- **Micro-Animations**: Hover states, buttons, sliding drawers, and alert banners transition smoothly using a standard `--transition-speed: 0.25s` cubic-bezier.
- **Monospace Typography**: All cryptographic keys, merchant tokens, MoMo transaction hashes, and response payloads use `Fira Code` to distinguish technical identifiers from normal content.

---

## 🔀 Portal Navigation & User Flows

```mermaid
graph TD
    A[Landing Page /] -->|Get Started| B[Register Page /portal/register.html]
    B -->|Submit Form| C{API Key Reveal Modal}
    C -->|Saved Key Checkbox| D[Dashboard /portal/dashboard.html]
    E[Login Page /portal/index.html] -->|Submit Key| D
    
    subgraph Merchant Portal [/portal/*]
        D --> Transactions[/portal/transactions.html]
        D --> DLQ[/portal/dlq.html]
        D --> Subscription[/portal/subscription.html]
    end

    subgraph Operator Control Room [/operator/*]
        O[Overview /operator/index.html] -->|Drilldown| M[Merchant Detail /operator/merchant.html]
        O --> SystemMetrics[System Metrics /operator/metrics.html]
    end
```

---

## 💻 Portal Pages Walkthrough

### 1. Ingestion Cockpit (`/portal/dashboard.html`)
- **4-Stat Metrics Ribbon**: Immediately tracks Month Calls, Delivery Success Rate, Total Money Volume, and Pending DLQ items.
- **Usage Warning Banners**: Proactively warns merchants using:
  - **80%+ Quota Warning**: Banner turns Amber advising a plan upgrade.
  - **100% Quota Blocked**: Banner turns Red, alerting that webhook deliveries are suspended until the next billing reset or plan upgrade.
- **Traffic Analytics**: Renders an inline SVG vector line chart highlighting ingestion peaks.

### 2. Transactions Log & Details Drawer (`/portal/transactions.html`)
- **Interactive Rows**: Clicking a transaction row slides out a detailed side-drawer.
- **Visual Timelines**: Illustrates step-by-step notification delivery phases:
  - *Webhook Received* &rarr; *Signature Verified* &rarr; *Payload Validated* &rarr; *Forwarding Completed*.

### 3. Dead Letter Queue & Retry Console (`/portal/dlq.html`)
- **Active Retries**: Developers can manually trigger retries.
- **Dynamic Polling**: Triggers a retry endpoint and polls for state transitions, showing active spinner statuses.
- **Celebrating Empty State**: Displays a clean visual space once all queued alerts have been resolved.

### 4. Subscription Quota Manager (`/portal/subscription.html`)
- Displays current month consumed ratios, remaining limits, and the next billing reset date (automatically calculated for the 1st of the next month).
- Renders a 3-month volume bar history using custom SVGs.
- Features a pricing matrix table with active plan rows highlighted and a mock upgrade checkout modal.

### 5. Operator Console (`/operator/index.html`)
- **Amber Industrial Theme**: Overridden variables visually mark the operator workspace.
- **Unified Actions**: Allows toggling status (Activate/Suspend) and adjusting quotas directly from the overview list or the detailed drill-down page (`/operator/merchant.html`).
- **Telemetry Monitor (`/operator/metrics.html`)**: Tracks uptime duration and database connection status alongside raw Prometheus copy-paste panels.

---

## 🛡️ Error Pages & System Integrity

- **404 Page (`/web/404.html`)**: Beautifully styled landing page for unmapped routes.
- **503 Page (`/web/503.html`)**: Industrial warning indicating database pool failures.
- **DB Health Middleware**:
  - The Gin router includes a middleware checking database connection health (`store.Ping()`) for all UI endpoints (`/`, `/portal/*`, `/operator/*`). If the database falls offline, it serves the `503.html` file with a `503 Service Unavailable` header automatically.
