// Observation only. Application state and interaction responses live on the server.
htmx.config.noSwap = [204, 304, '5xx'];

const starts = new WeakMap();
let requestCount = 0;

function textElement(tag, className, text) {
  const element = document.createElement(tag);
  element.className = className;
  element.textContent = text;
  return element;
}

document.addEventListener('htmx:before:request', (event) => {
  starts.set(event.detail.ctx, performance.now());
});

document.addEventListener('htmx:after:request', (event) => {
  const ctx = event.detail.ctx;
  const log = document.getElementById('wire-log');
  if (!log) return;
  log.querySelector('.wire-empty')?.remove();
  const entry = document.createElement('details');
  entry.className = 'request-entry';
  const summary = document.createElement('summary');
  const url = new URL(ctx.request.action, location.href);
  const elapsed = Math.round(performance.now() - (starts.get(ctx) ?? performance.now()));
  summary.append(
    textElement('span', 'request-method', ctx.request.method.toUpperCase()),
    textElement('span', 'request-url', url.pathname + url.search),
    textElement('span', 'request-status', String(ctx.response.status)),
    textElement('span', '', `${elapsed} ms`),
  );
  const headers = [...ctx.response.headers].map(([name, value]) => `${name}: ${value}`).join('\n');
  entry.append(summary, textElement('pre', '', `${headers}\n\n${ctx.text.slice(0, 20000)}${ctx.text.length > 20000 ? '\n…response truncated for display' : ''}`));
  log.prepend(entry);
  while (log.children.length > 12) log.lastElementChild.remove();
  document.getElementById('wire-count').textContent = `${++requestCount} requests`;
  if (ctx.response.status >= 500) {
    document.getElementById('notice').textContent = 'The server could not complete that request. Your current view is unchanged; try again.';
  }
});

document.addEventListener('htmx:error', () => {
  document.getElementById('notice').textContent = 'The request failed. Check your connection and try again.';
});

document.getElementById('clear-wire')?.addEventListener('click', () => {
  document.getElementById('wire-log').replaceChildren(textElement('p', 'wire-empty', 'Log cleared. Make another move.'));
  document.getElementById('wire-count').textContent = '0 requests';
  requestCount = 0;
});
