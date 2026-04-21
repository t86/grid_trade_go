function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function formatDate(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString();
}

async function loadDashboard() {
  const response = await fetch("/debug/dashboard");
  if (!response.ok) {
    throw new Error(`dashboard ${response.status}`);
  }
  return response.json();
}

async function loadAccounts(market) {
  const response = await fetch(`/debug/accounts?market=${encodeURIComponent(market)}`);
  if (!response.ok) {
    throw new Error(`accounts ${response.status}`);
  }
  return response.json();
}

function renderAlerts(alerts) {
  const root = document.getElementById("alerts");
  if (!alerts || alerts.length === 0) {
    root.innerHTML = '<div class="empty">No active alerts</div>';
    return;
  }

  root.innerHTML = alerts.map((alert) => `
    <article class="alert ${escapeHTML(alert.severity)}">
      <div class="alerts-head">
        <strong>${escapeHTML(alert.title)}</strong>
        <span>${escapeHTML(alert.market)} / ${escapeHTML(alert.account || "system")}</span>
      </div>
      <p>${escapeHTML(alert.detail || "")}</p>
    </article>
  `).join("");
}

function renderMarkets(markets) {
  const root = document.getElementById("markets");
  root.innerHTML = markets.map((market) => `
    <section class="market" data-market="${escapeHTML(market.market)}">
      <div class="market-head">
        <div>
          <p class="eyebrow">${escapeHTML(market.market)}</p>
          <h2>${market.market === "spot" ? "Spot" : "Futures"}</h2>
        </div>
        <span class="badge ${escapeHTML(market.health)}">${escapeHTML(market.health)}</span>
      </div>
      <div class="metrics">
        <div class="metric"><div class="metric-label">online accounts</div><div class="metric-value">${market.onlineAccounts ?? 0}</div></div>
        <div class="metric"><div class="metric-label">degraded accounts</div><div class="metric-value">${market.degradedAccounts ?? 0}</div></div>
        <div class="metric"><div class="metric-label">heartbeat lag</div><div class="metric-value">${market.avgHeartbeatLagMs ?? 0}ms</div></div>
        <div class="metric"><div class="metric-label">listen key healthy</div><div class="metric-value">${market.listenKeyHealthyAccounts ?? 0}</div></div>
        <div class="metric"><div class="metric-label">reduce-only accounts</div><div class="metric-value">${market.reduceOnlyAccounts ?? 0}</div></div>
        <div class="metric"><div class="metric-label">last reconnect</div><div class="metric-value">${escapeHTML(formatDate(market.lastReconnectAt))}</div></div>
      </div>
      <div class="accounts">
        <details>
          <summary>View Accounts</summary>
          <div class="account-list empty">Loading...</div>
        </details>
      </div>
    </section>
  `).join("");

  root.querySelectorAll(".market details").forEach((details) => {
    details.addEventListener("toggle", async () => {
      if (!details.open) return;
      const section = details.closest(".market");
      const market = section.dataset.market;
      const container = section.querySelector(".account-list");
      if (container.dataset.loaded === "true") return;
      try {
        const payload = await loadAccounts(market);
        container.innerHTML = renderAccountRows(payload.accounts || []);
        container.dataset.loaded = "true";
      } catch (error) {
        container.innerHTML = `<div class="empty">${escapeHTML(error.message)}</div>`;
      }
    });
  });
}

function renderAccountRows(accounts) {
  if (!accounts || accounts.length === 0) {
    return '<div class="empty">No accounts</div>';
  }
  return accounts.map((account) => `
    <article class="account-row">
      <div>
        <strong>${escapeHTML(account.account)}</strong>
        <div class="account-meta">${escapeHTML(account.sessionState)} / listen key ${escapeHTML(account.listenKeyState)}</div>
        <div class="account-meta">${escapeHTML(account.lastError || "no recent error")}</div>
      </div>
      <div class="account-meta">reduce-only: ${account.reduceOnly ? "yes" : "no"}</div>
      <div class="account-meta">backoff: ${account.activeBackoff ? "on" : "off"}</div>
    </article>
  `).join("");
}

async function refresh() {
  const refreshState = document.getElementById("refresh-state");
  try {
    const dashboard = await loadDashboard();
    renderAlerts(dashboard.alerts || []);
    renderMarkets(dashboard.markets || []);
    refreshState.textContent = `最近刷新 ${new Date().toLocaleTimeString()}`;
  } catch (error) {
    document.getElementById("alerts").innerHTML = `<div class="empty">${escapeHTML(error.message)}</div>`;
    refreshState.textContent = `刷新失败: ${error.message}`;
  }
}

refresh();
setInterval(refresh, 2000);
