// Local UI behavior only: native dialog focus, selection feedback, and HTTP inspection.
// Tasks, validation, filters, history URLs, and job progress are server-owned.
// 422 remains swappable: its HTML is the validation interface, not a transport error.
htmx.config.noSwap = [204, 304, 400, 403, 404, 405, 413, '5xx'];

const starts = new WeakMap();
const initializedDialogs = new WeakSet();
let requestCount = 0;
let editorWasOpen = false;
let returnFocusID;

function textElement(tag, className, text) {
  const element = document.createElement(tag);
  element.className = className;
  element.textContent = text;
  return element;
}

function initializeUI() {
  const editor = document.getElementById('editor');
  if (editor && !initializedDialogs.has(editor)) {
    initializedDialogs.add(editor);
    editorWasOpen = true;
    returnFocusID ??= editor.dataset.taskId === '-1' ? 'new-task' : `task-${editor.dataset.taskId}`;
    editor.removeAttribute('open');
    editor.showModal();
    editor.addEventListener('cancel', event => {
      event.preventDefault();
      editor.querySelector('[data-close-editor]').click();
    });
    editor.querySelector('[autofocus]').focus({ preventScroll: true });
  } else if (!editor && editorWasOpen) {
    editorWasOpen = false;
    (document.getElementById(returnFocusID) ?? document.getElementById('view-heading'))?.focus({ preventScroll: true });
    returnFocusID = undefined;
  }
  updateSelection();
}

function updateSelection() {
  const counter = document.getElementById('selection-count');
  if (!counter) return;
  const count = document.querySelectorAll('.list-form input[name="task"]:checked').length;
  counter.textContent = count ? `${count} selected` : 'Select tasks to complete together';
}

function showError(message) {
  const editor = document.querySelector('.editor-body');
  if (editor) {
    let error = editor.querySelector('.error');
    if (!error) {
      error = textElement('p', 'error', '');
      error.setAttribute('role', 'alert');
      editor.prepend(error);
    }
    error.textContent = message;
  } else {
    document.getElementById('notice').textContent = message;
  }
}

document.addEventListener('htmx:before:request', event => {
  const ctx = event.detail.ctx;
  starts.set(ctx, performance.now());
  if (ctx.sourceElement.matches('.task-card, td a, #new-task')) returnFocusID = ctx.sourceElement.id;
});

document.addEventListener('htmx:after:swap', initializeUI);
document.addEventListener('change', updateSelection);

document.addEventListener('htmx:after:request', event => {
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
    textElement('span', 'request-time', `${elapsed} ms`),
  );
  const headers = [...ctx.response.headers].map(([name, value]) => `${name}: ${value}`).join('\n');
  entry.append(summary, textElement('pre', '', `${headers}\n\n${ctx.text.slice(0, 20000)}${ctx.text.length > 20000 ? '\n…response truncated for display' : ''}`));
  log.prepend(entry);
  while (log.children.length > 12) log.lastElementChild.remove();
  document.getElementById('wire-count').textContent = `${++requestCount} requests`;
  if (ctx.response.status === 404) {
    showError('This item no longer exists. Close the editor and refresh the workspace.');
  } else if (ctx.response.status >= 400 && ctx.response.status !== 422) {
    showError('The server could not complete the request. Your current view is unchanged; try again.');
  }
});

document.addEventListener('htmx:error', event => {
  // hx-sync intentionally aborts obsolete searches; that is not a connection failure.
  if (event.detail.error?.name === 'AbortError') return;
  showError('The request failed. Check your connection and try again.');
});

document.addEventListener('click', event => {
  if (event.target.closest('#clear-wire')) {
    document.getElementById('wire-log').replaceChildren(textElement('p', 'wire-empty', 'Log cleared. Make another move.'));
    document.getElementById('wire-count').textContent = '0 requests';
    requestCount = 0;
  }
});

initializeUI();
