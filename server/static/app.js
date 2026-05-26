// Renders the /api/status snapshot. Note: alert and forecast text is injected
// via innerHTML. The only data sources are NWS and MTA (institutional APIs) and
// this is a single-user self-hosted dashboard, so the injection risk is low. If
// you expose this on a shared network, switch to textContent or sanitize first.
const REFRESH_MS = 30000;

function fmtTime(iso) {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

function statusClass(status) {
  if (!status || status === "On time" || status === "Good Service") return "ok";
  if (status.startsWith("Delayed") || status === "Delays") return "bad";
  return "warn";
}

function renderWeather(w) {
  const el = document.getElementById("weather");
  if (w.err) { el.innerHTML = `<span class="err">Weather unavailable: ${w.err}</span>`; return; }
  let html = `<strong>${w.summary || "-"}</strong> &middot; ${w.tempF}&deg;F &middot; ${w.precipChance}% precip`;
  if (w.alerts && w.alerts.length) {
    html += w.alerts.map(a => `<div class="bad">&#9888; ${a.event}: ${a.headline}</div>`).join("");
  }
  el.innerHTML = html;
}

function renderTrainLeg(id, leg) {
  const el = document.getElementById(id);
  const label = `Metro-North ${leg.origin} to ${leg.dest}`;
  if (leg.err) { el.innerHTML = `${label}: <span class="err">${leg.err}</span>`; return; }
  if (!leg.trains || !leg.trains.length) { el.innerHTML = `${label}: <span class="muted">no upcoming trains</span>`; return; }
  const rows = leg.trains.map(t => {
    const track = t.track ? `<span class="track">track ${t.track}</span>` : "";
    return `<div class="train"><span class="time">${fmtTime(t.departure)}</span>` +
      `<span class="${statusClass(t.status)}">${t.status}</span>${track}</div>`;
  }).join("");
  el.innerHTML = `${label} <span class="muted">(${leg.source})</span><div class="trains">${rows}</div>`;
}

function renderSubway(id, sub) {
  const el = document.getElementById(id);
  if (sub.err) { el.innerHTML = `subway: <span class="err">${sub.err}</span>`; return; }
  let html = `subway: <span class="${statusClass(sub.status)}">${sub.status}</span>`;
  if (sub.alerts && sub.alerts.length) {
    html += sub.alerts.map(a => `<div class="warn">${a}</div>`).join("");
  }
  el.innerHTML = html;
}

function renderDrive(id, drive) {
  const el = document.getElementById(id);
  if (drive.err) { el.innerHTML = `Drive: <span class="err">${drive.err}</span>`; return; }
  el.innerHTML = `Drive home &harr; station: <strong>${drive.durationMin} min</strong>`;
}

async function refresh() {
  try {
    const res = await fetch("/api/status");
    const snap = await res.json();
    renderWeather(snap.weather);
    renderDrive("out-drive", snap.drive);
    renderTrainLeg("out-train", snap.outbound);
    renderSubway("out-subway", snap.subway);
    renderSubway("in-subway", snap.subway);
    renderTrainLeg("in-train", snap.inbound);
    renderDrive("in-drive", snap.drive);
    document.getElementById("freshness").textContent =
      "updated " + fmtTime(snap.generatedAt);
  } catch (e) {
    document.getElementById("freshness").textContent = "refresh failed";
  }
}

refresh();
setInterval(refresh, REFRESH_MS);
