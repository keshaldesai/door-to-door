// Renders the /api/status snapshot. Note: alert and forecast text is injected
// via innerHTML. The only data sources are NWS and MTA (institutional APIs) and
// this is a single-user self-hosted dashboard, so the injection risk is low. If
// you expose this on a shared network, switch to textContent or sanitize first.
const REFRESH_MS = 30000;

function fmtTime(iso) {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

function fmtUntil(ms) {
  const m = Math.max(0, Math.round(ms / 60000));
  if (m < 60) return `${m}m`;
  return `${Math.floor(m / 60)}h ${m % 60}m`;
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
  // Go marshals zero time.Time as "0001-01-01T00:00:00Z"; that parses to a
  // negative epoch, so the getTime() > 0 check is what actually gates display.
  const rise = new Date(w.sunriseAt);
  const set = new Date(w.sunsetAt);
  if (rise.getTime() > 0 && set.getTime() > 0) {
    const now = new Date();
    const next = [rise, set].filter(d => d > now).sort((a, b) => a - b)[0];
    let hint = "";
    if (next) {
      const ms = next - now;
      if (ms <= 2 * 60 * 60 * 1000) {
        const label = next === rise ? "sunrise" : "sunset";
        hint = ` <span class="muted">(${label} in ${fmtUntil(ms)})</span>`;
      }
    }
    html += `<div class="muted">Sunrise ${fmtTime(w.sunriseAt)}, sunset ${fmtTime(w.sunsetAt)}${hint}</div>`;
  }
  el.innerHTML = html;
}

function renderTrainLeg(id, leg) {
  const el = document.getElementById(id);
  const label = `Metro-North ${leg.origin} to ${leg.dest}`;
  if (leg.err) { el.innerHTML = `${label}: <span class="err">${leg.err}</span>`; return; }
  if (!leg.trains || !leg.trains.length) { el.innerHTML = `${label}: <span class="muted">no upcoming trains</span>`; return; }
  const offsetMin = leg.leaveOffsetMin || 0;
  const expected = leg.expectedTrack || "";
  const rows = leg.trains.map(t => {
    const leaveAt = offsetMin > 0
      ? new Date(new Date(t.departure).getTime() - offsetMin * 60_000)
      : null;
    const leave = leaveAt ? `<span class="leave">leave ${fmtTime(leaveAt.toISOString())}</span>` : "";
    let track;
    if (!t.track) {
      track = `<span class="track muted">track TBD</span>`;
    } else if (expected && t.track !== expected) {
      track = `<span class="track bad">track ${t.track} (expected ${expected})</span>`;
    } else {
      track = `<span class="track">track ${t.track}</span>`;
    }
    return `<div class="train"><span class="time">${fmtTime(t.departure)}</span>` +
      `<span class="${statusClass(t.status)}">${t.status}</span>${leave}${track}</div>`;
  }).join("");
  el.innerHTML = `${label} <span class="muted">(${leg.source})</span><div class="trains">${rows}</div>`;
}

function renderSubway(id, sub, countdown) {
  const el = document.getElementById(id);
  const label = sub.line ? `${sub.line} train` : "Subway";
  if (sub.err) { el.innerHTML = `${label}: <span class="err">${sub.err}</span>`; return; }
  let html = `${label}: <span class="${statusClass(sub.status)}">${sub.status}</span>`;
  if (sub.alerts && sub.alerts.length) {
    html += sub.alerts.map(a => `<div class="warn">${a}</div>`).join("");
  }
  if (countdown) {
    if (countdown.err) {
      html += `<div class="err">Countdown: ${countdown.err}</div>`;
    } else if (countdown.arrivals && countdown.arrivals.length) {
      const now = Date.now();
      const mins = countdown.arrivals
        .map(a => Math.max(0, Math.round((new Date(a).getTime() - now) / 60000)))
        .map(m => `${m} min`)
        .join(", ");
      const dir = directionLabel(countdown.stopId);
      html += `<div class="muted">Next at ${countdown.stopId}${dir ? ` (${dir})` : ""}: ${mins}</div>`;
    }
  }
  el.innerHTML = html;
}

function directionLabel(stopId) {
  if (typeof stopId === "string" && stopId.length > 0) {
    const last = stopId[stopId.length - 1];
    if (last === "N") return "uptown";
    if (last === "S") return "downtown";
  }
  return "";
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
    renderSubway("out-subway", snap.subway, snap.outboundSubway);
    renderSubway("in-subway", snap.subway, snap.inboundSubway);
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
