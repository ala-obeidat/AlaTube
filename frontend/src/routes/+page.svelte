<script lang="ts">
  import { onMount } from 'svelte';
  import { analyze, apiURL, createJob, readableBytes, withToken, type Analysis, type ApiError, type Format, type JobEvent } from '$lib/api';
  import { readPendingShare } from '$lib/share';

  type Theme = 'light' | 'dark';

  let url = '';
  let analysis: Analysis | null = null;
  let selectedVideo = '';
  let selectedAudio = '';
  let busy = false;
  let error = '';
  let jobEvent: JobEvent | null = null;
  let theme: Theme = 'light';

  $: videoFormats = analysis?.formats.filter((f) => f.kind === 'video' || f.kind === 'muxed') ?? [];
  $: audioFormats = analysis?.formats.filter((f) => f.kind === 'audio') ?? [];
  $: progressPct = jobEvent ? Math.round(jobEvent.progress * 100) : 0;

  onMount(async () => {
    theme = (document.documentElement.getAttribute('data-theme') as Theme) || 'light';

    if ('serviceWorker' in navigator) {
      await navigator.serviceWorker.register('/service-worker.js');
    }
    const pending = await readPendingShare().catch(() => null);
    if (pending?.text) {
      url = pending.text;
      await runAnalyze();
    }
  });

  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', theme);
    try {
      localStorage.setItem('alatube-theme', theme);
    } catch {
      /* ignore */
    }
  }

  async function runAnalyze() {
    busy = true;
    error = '';
    jobEvent = null;
    try {
      analysis = await analyze(url);
      selectedVideo = chooseDefaultVideo(videoFormats)?.formatId ?? '';
      selectedAudio = audioFormats[0]?.formatId ?? '';
    } catch (err) {
      error = messageFor(err);
      analysis = null;
    } finally {
      busy = false;
    }
  }

  async function runJob() {
    if (!analysis || !selectedVideo) return;
    busy = true;
    error = '';
    try {
      const job = await createJob(analysis.videoId, selectedVideo, selectedAudio || undefined);
      watchJob(withToken(apiURL(job.eventsUrl)));
    } catch (err) {
      error = messageFor(err);
    } finally {
      busy = false;
    }
  }

  function watchJob(eventsUrl: string) {
    const events = new EventSource(eventsUrl, { withCredentials: true });
    events.addEventListener('job', (event) => {
      jobEvent = JSON.parse((event as MessageEvent).data);
      if (jobEvent?.state === 'completed' || jobEvent?.state === 'failed' || jobEvent?.state === 'expired') {
        events.close();
      }
    });
    events.onerror = () => {
      error = 'The job progress connection was interrupted. Refresh and try again if the job does not finish.';
      events.close();
    };
  }

  function chooseDefaultVideo(formats: Format[]) {
    return [...formats]
      .filter((f) => !f.height || f.height <= 720)
      .sort((a, b) => (b.height ?? 0) - (a.height ?? 0))[0] ?? formats[0];
  }

  function messageFor(err: unknown) {
    const api = err as ApiError;
    if (api?.error?.code === 'request_failed' || api?.error?.code === 'media_analysis_failed') {
      return 'Could not reach the API. Try again in a moment.';
    }
    return api?.error?.message ?? 'Something went wrong.';
  }

  function formatDuration(s?: number): string {
    if (!s) return 'Duration unavailable';
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = s % 60;
    return h > 0 ? `${h}h ${m}m` : m > 0 ? `${m}m ${sec}s` : `${sec}s`;
  }
</script>

<svelte:head>
  <title>AlaTube</title>
</svelte:head>

<header class="topbar">
  <div class="brand">
    <span class="logo" aria-hidden="true">▶</span>
    <span class="brand-name">AlaTube</span>
  </div>
  <button
    class="theme-toggle"
    type="button"
    on:click={toggleTheme}
    aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
    title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
  >
    {#if theme === 'dark'}
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
      </svg>
    {:else}
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M21 12.79A9 9 0 1 1 11.21 3a7 7 0 0 0 9.79 9.79z" />
      </svg>
    {/if}
  </button>
</header>

<main class="shell">
  <section class="panel" aria-labelledby="title">
    <div class="hero">
      <p class="eyebrow">Private media analysis</p>
      <h1 id="title">Save any YouTube video</h1>
      <p class="lead">Paste a link, pick a quality, download the file. Nothing leaves the server except the request to YouTube.</p>
    </div>

    <form on:submit|preventDefault={runAnalyze} class="analyze-form">
      <label for="url" class="field-label">YouTube URL</label>
      <div class="input-row">
        <input
          id="url"
          type="url"
          bind:value={url}
          autocomplete="off"
          autocapitalize="off"
          autocorrect="off"
          spellcheck="false"
          inputmode="url"
          placeholder="https://www.youtube.com/watch?v=…"
        />
        <button class="primary" type="submit" disabled={busy || !url.trim()}>
          {#if busy}
            <span class="spinner" aria-hidden="true"></span>
            <span>Working</span>
          {:else}
            <span>Analyze</span>
          {/if}
        </button>
      </div>
    </form>

    {#if error}
      <p class="error" role="alert">{error}</p>
    {/if}

    {#if analysis}
      <article class="result">
        {#if analysis.thumbnailUrl}
          <img src={analysis.thumbnailUrl} alt="" loading="lazy" />
        {/if}
        <div class="result-meta">
          <h2>{analysis.title || analysis.videoId}</h2>
          <p class="muted">{formatDuration(analysis.durationSeconds)}</p>
        </div>
      </article>

      <div class="selectors">
        <label class="field-label">
          Video quality
          <div class="select-wrap">
            <select bind:value={selectedVideo}>
              {#each videoFormats as format}
                <option value={format.formatId}>
                  {format.height ? `${format.height}p` : format.formatId} · {format.container ?? 'media'} · {readableBytes(format.estimatedBytes)}
                </option>
              {/each}
            </select>
          </div>
        </label>

        <label class="field-label">
          Audio
          <div class="select-wrap">
            <select bind:value={selectedAudio}>
              <option value="">Auto (use muxed audio if available)</option>
              {#each audioFormats as format}
                <option value={format.formatId}>
                  {format.container ?? format.formatId} · {readableBytes(format.estimatedBytes)}
                </option>
              {/each}
            </select>
          </div>
        </label>
      </div>

      <button class="primary block" on:click={runJob} disabled={busy || !selectedVideo}>
        {#if busy}
          <span class="spinner" aria-hidden="true"></span>
          <span>Working</span>
        {:else}
          <span>Create download</span>
        {/if}
      </button>
    {/if}

    {#if jobEvent}
      <section class="status" aria-live="polite">
        <div class="meter" role="progressbar" aria-valuenow={progressPct} aria-valuemin="0" aria-valuemax="100">
          <span style={`width: ${Math.max(4, progressPct)}%`}></span>
        </div>
        <div class="status-row">
          <p class="status-text">{jobEvent.message}</p>
          <span class="status-pct">{progressPct}%</span>
        </div>
        {#if jobEvent.downloadUrl}
          <a class="download block" href={withToken(apiURL(jobEvent.downloadUrl))}>
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3" />
            </svg>
            <span>Download file</span>
          </a>
        {/if}
        {#if jobEvent.error}
          <p class="error" role="alert">{jobEvent.error.message}</p>
        {/if}
      </section>
    {/if}
  </section>

  <footer class="footer">
    <p>Private use only · Respect creators' rights and YouTube's Terms of Service.</p>
  </footer>
</main>

<style>
  :global(:root) {
    color-scheme: light;
    --bg: #f6f7f9;
    --surface: #ffffff;
    --surface-2: #f1f3f5;
    --text: #0f1417;
    --text-muted: #5b6770;
    --border: #e1e5ea;
    --border-strong: #ced3d9;
    --accent: #0b5c6b;
    --accent-hover: #094851;
    --accent-soft: #e0eef0;
    --accent-on: #ffffff;
    --error: #b3261e;
    --error-bg: #fdecea;
    --shadow-sm: 0 1px 2px rgba(15, 20, 23, 0.05);
    --shadow-md: 0 8px 24px rgba(15, 20, 23, 0.08);
    --radius: 10px;
    --radius-sm: 6px;
  }

  :global([data-theme='dark']) {
    color-scheme: dark;
    --bg: #0a0e14;
    --surface: #111827;
    --surface-2: #1a2331;
    --text: #e5e9ee;
    --text-muted: #98a2af;
    --border: #243044;
    --border-strong: #324159;
    --accent: #14b8a6;
    --accent-hover: #2dd4bf;
    --accent-soft: #103937;
    --accent-on: #06241f;
    --error: #ff8b87;
    --error-bg: #2a1313;
    --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.4);
    --shadow-md: 0 8px 24px rgba(0, 0, 0, 0.5);
  }

  @media (prefers-color-scheme: dark) {
    :global(:root:not([data-theme='light'])) {
      color-scheme: dark;
      --bg: #0a0e14;
      --surface: #111827;
      --surface-2: #1a2331;
      --text: #e5e9ee;
      --text-muted: #98a2af;
      --border: #243044;
      --border-strong: #324159;
      --accent: #14b8a6;
      --accent-hover: #2dd4bf;
      --accent-soft: #103937;
      --accent-on: #06241f;
      --error: #ff8b87;
      --error-bg: #2a1313;
      --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.4);
      --shadow-md: 0 8px 24px rgba(0, 0, 0, 0.5);
    }
  }

  :global(*),
  :global(*::before),
  :global(*::after) {
    box-sizing: border-box;
  }

  :global(body) {
    margin: 0;
    min-height: 100dvh;
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    color: var(--text);
    background: var(--bg);
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    transition: background-color 180ms ease, color 180ms ease;
  }

  .topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 24px;
    max-width: 880px;
    margin: 0 auto;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    font-weight: 800;
    font-size: 1.05rem;
    letter-spacing: -0.01em;
  }

  .logo {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    background: var(--accent);
    color: var(--accent-on);
    display: grid;
    place-items: center;
    font-size: 0.95rem;
  }

  .brand-name {
    color: var(--text);
  }

  .theme-toggle {
    width: 40px;
    height: 40px;
    border-radius: 999px;
    background: var(--surface);
    color: var(--text);
    border: 1px solid var(--border);
    cursor: pointer;
    display: grid;
    place-items: center;
    transition: background-color 150ms ease, border-color 150ms ease, transform 150ms ease;
  }

  .theme-toggle:hover {
    background: var(--surface-2);
    border-color: var(--border-strong);
  }

  .theme-toggle:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  .shell {
    max-width: 760px;
    margin: 0 auto;
    padding: 8px 24px 48px;
  }

  .panel {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 16px;
    box-shadow: var(--shadow-md);
    padding: 32px;
    display: grid;
    gap: 24px;
  }

  .hero {
    display: grid;
    gap: 6px;
  }

  .eyebrow {
    margin: 0;
    color: var(--accent);
    font-size: 0.78rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  h1 {
    margin: 0;
    font-size: 2rem;
    font-weight: 800;
    letter-spacing: -0.02em;
    color: var(--text);
  }

  h2 {
    margin: 0 0 4px;
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--text);
  }

  .lead {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.95rem;
    line-height: 1.5;
  }

  .field-label {
    display: grid;
    gap: 8px;
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
    letter-spacing: 0.01em;
  }

  .analyze-form,
  .selectors,
  .status {
    display: grid;
    gap: 14px;
  }

  .input-row {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 10px;
  }

  input,
  select {
    width: 100%;
    min-height: 48px;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
    padding: 0 14px;
    border-radius: var(--radius);
    font: inherit;
    box-shadow: var(--shadow-sm);
    transition: border-color 150ms ease, box-shadow 150ms ease;
  }

  input::placeholder {
    color: var(--text-muted);
  }

  input:focus-visible,
  select:focus-visible {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 4px var(--accent-soft);
  }

  .select-wrap {
    position: relative;
  }

  .select-wrap select {
    appearance: none;
    -webkit-appearance: none;
    padding-right: 40px;
    cursor: pointer;
  }

  .select-wrap::after {
    content: '';
    position: absolute;
    right: 16px;
    top: 50%;
    width: 8px;
    height: 8px;
    border-right: 2px solid var(--text-muted);
    border-bottom: 2px solid var(--text-muted);
    transform: translateY(-70%) rotate(45deg);
    pointer-events: none;
  }

  button,
  .download {
    min-height: 48px;
    border: 0;
    padding: 0 22px;
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--accent-on);
    font: inherit;
    font-weight: 700;
    cursor: pointer;
    text-decoration: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    transition: background-color 150ms ease, transform 100ms ease;
  }

  button:hover:not(:disabled),
  .download:hover {
    background: var(--accent-hover);
  }

  button:active:not(:disabled) {
    transform: translateY(1px);
  }

  button:focus-visible,
  .download:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  button:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .block {
    width: 100%;
    justify-self: stretch;
  }

  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid currentColor;
    border-right-color: transparent;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .error {
    margin: 0;
    padding: 12px 14px;
    background: var(--error-bg);
    color: var(--error);
    border-radius: var(--radius);
    font-size: 0.9rem;
    font-weight: 600;
  }

  .result {
    display: grid;
    grid-template-columns: 160px 1fr;
    gap: 16px;
    align-items: center;
    padding: 16px;
    border: 1px solid var(--border);
    background: var(--surface-2);
    border-radius: var(--radius);
  }

  .result img {
    width: 160px;
    aspect-ratio: 16 / 9;
    object-fit: cover;
    border-radius: var(--radius-sm);
    background: var(--surface);
  }

  .result-meta {
    display: grid;
    gap: 4px;
  }

  .muted {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .status {
    padding: 16px;
    background: var(--surface-2);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .status-row {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
  }

  .status-text {
    margin: 0;
    color: var(--text);
    font-weight: 600;
  }

  .status-pct {
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
    font-size: 0.9rem;
  }

  .meter {
    height: 10px;
    overflow: hidden;
    border-radius: 999px;
    background: var(--border);
  }

  .meter span {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--accent), var(--accent-hover));
    transition: width 350ms cubic-bezier(0.22, 0.61, 0.36, 1);
  }

  .footer {
    margin-top: 24px;
    text-align: center;
  }

  .footer p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.8rem;
  }

  @media (max-width: 640px) {
    .topbar {
      padding: 12px 16px;
    }

    .shell {
      padding: 4px 16px 32px;
    }

    .panel {
      padding: 20px;
      border-radius: 14px;
    }

    h1 {
      font-size: 1.6rem;
    }

    .lead {
      font-size: 0.9rem;
    }

    .input-row {
      grid-template-columns: 1fr;
    }

    .result {
      grid-template-columns: 1fr;
      padding: 12px;
    }

    .result img {
      width: 100%;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    :global(body),
    button,
    .theme-toggle,
    input,
    select,
    .meter span {
      transition: none;
    }
    .spinner {
      animation: none;
    }
  }
</style>
