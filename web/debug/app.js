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

async function loadEvents(market, account) {
  const response = await fetch(`/debug/events?market=${encodeURIComponent(market)}&account=${encodeURIComponent(account || "")}`);
  if (!response.ok) {
    throw new Error(`events ${response.status}`);
  }
  return response.json();
}

function renderAlerts(alerts) {
  const root = document.getElementById("alerts");
  if (!alerts || alerts.length === 0) {
    root.innerHTML = '<div class="empty">暂无活跃告警</div>';
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
          <p class="eyebrow">${escapeHTML(labelMarketCode(market.market))}</p>
          <h2>${escapeHTML(labelMarket(market.market))}</h2>
        </div>
        <span class="badge ${escapeHTML(market.health)}">${escapeHTML(labelHealth(market.health))}</span>
      </div>
      <div class="metrics">
        <div class="metric"><div class="metric-label">在线账户</div><div class="metric-value">${market.onlineAccounts ?? 0}</div></div>
        <div class="metric"><div class="metric-label">降级账户</div><div class="metric-value">${market.degradedAccounts ?? 0}</div></div>
        <div class="metric"><div class="metric-label">心跳延迟</div><div class="metric-value">${market.avgHeartbeatLagMs ?? 0}ms</div></div>
        <div class="metric"><div class="metric-label">Listen Key 正常</div><div class="metric-value">${market.listenKeyHealthyAccounts ?? 0}</div></div>
        <div class="metric"><div class="metric-label">只减仓账户</div><div class="metric-value">${market.reduceOnlyAccounts ?? 0}</div></div>
        <div class="metric"><div class="metric-label">最近重连</div><div class="metric-value">${escapeHTML(formatDate(market.lastReconnectAt))}</div></div>
      </div>
      <div class="accounts">
        <details>
          <summary>查看账户明细</summary>
          <div class="account-list empty">加载中...</div>
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
    return '<div class="empty">暂无账户</div>';
  }
  return accounts.map((account) => `
    <article class="account-row">
      <div>
        <strong>${escapeHTML(account.account)}</strong>
        <div class="account-meta">会话状态：${escapeHTML(labelSessionState(account.sessionState))} / Listen Key：${escapeHTML(labelListenKeyState(account.listenKeyState))}</div>
        <div class="account-meta">密钥引用：${escapeHTML(account.secretRef || "-")}</div>
        <div class="account-meta">密钥状态：${escapeHTML(labelSecretStatus(account.secretStatus))}</div>
        <div class="account-meta">最近心跳：${escapeHTML(formatDate(account.lastHeartbeatAt))}</div>
        <div class="account-meta">最近重连：${escapeHTML(formatDate(account.lastReconnectAt))}</div>
        <div class="account-meta">Listen Key 到期：${escapeHTML(formatDate(account.listenKeyExpiresAt))}</div>
        <div class="account-meta">${escapeHTML(account.lastError || "暂无最近错误")}</div>
      </div>
      <div class="account-meta">只减仓：${account.reduceOnly ? "是" : "否"}</div>
      <div class="account-meta">退避：${account.activeBackoff ? "开启" : "关闭"}</div>
    </article>
  `).join("");
}

function renderEvents(events) {
  const root = document.getElementById("events");
  if (!events || events.length === 0) {
    root.innerHTML = '<div class="empty">暂无最近事件</div>';
    return;
  }
  root.innerHTML = events.map((event) => `
    <article class="event">
      <strong>${escapeHTML(event.message)}</strong>
    </article>
  `).join("");
}

function labelHealth(value) {
  if (value === "healthy") return "正常";
  if (value === "warning") return "警告";
  if (value === "degraded") return "降级";
  return value || "-";
}

function labelMarket(value) {
  if (value === "spot") return "现货";
  if (value === "futures_um") return "合约";
  return value || "-";
}

function labelMarketCode(value) {
  if (value === "spot") return "SPOT";
  if (value === "futures_um") return "FUTURES UM";
  return value || "-";
}

function labelSessionState(value) {
  if (value === "connecting") return "连接中";
  if (value === "active") return "运行中";
  if (value === "degraded") return "已降级";
  if (value === "backing_off") return "退避中";
  if (value === "banned_until") return "封禁中";
  return value || "-";
}

function labelListenKeyState(value) {
  if (value === "creating") return "创建中";
  if (value === "healthy") return "正常";
  if (value === "stale") return "已过期";
  if (value === "expiring") return "即将过期";
  if (value === "not_configured") return "未配置";
  return value || "-";
}

function labelSecretStatus(value) {
  if (value === "loaded") return "已加载";
  if (value === "load_failed") return "加载失败";
  if (value === "not_configured") return "未配置";
  return value || "-";
}

async function refresh() {
  const refreshState = document.getElementById("refresh-state");
  try {
    const dashboard = await loadDashboard();
    const eventsPayload = await loadEvents("futures_um", "primary");
    renderAlerts(dashboard.alerts || []);
    renderMarkets(dashboard.markets || []);
    renderEvents(eventsPayload.events || []);
    refreshState.textContent = `最近刷新 ${new Date().toLocaleTimeString()}`;
  } catch (error) {
    document.getElementById("alerts").innerHTML = `<div class="empty">${escapeHTML(error.message)}</div>`;
    document.getElementById("events").innerHTML = `<div class="empty">${escapeHTML(error.message)}</div>`;
    refreshState.textContent = `刷新失败: ${error.message}`;
  }
}

refresh();
setInterval(refresh, 2000);
