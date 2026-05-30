<script lang="ts">
  import { onMount } from 'svelte';
  import { analyze, apiURL, createJob, readableBytes, type Analysis, type ApiError, type Format, type JobEvent } from '$lib/api';
  import { readPendingShare } from '$lib/share';

  let url = '';
  let analysis: Analysis | null = null;
  let selectedVideo = '';
  let selectedAudio = '';
  let busy = false;
  let error = '';
  let jobEvent: JobEvent | null = null;

  $: videoFormats = analysis?.formats.filter((f) => f.kind === 'video' || f.kind === 'muxed') ?? [];
  $: audioFormats = analysis?.formats.filter((f) => f.kind === 'audio') ?? [];

  onMount(async () => {
    if ('serviceWorker' in navigator) {
      await navigator.serviceWorker.register('/service-worker.js');
    }
    const pending = await readPendingShare().catch(() => null);
    if (pending?.text) {
      url = pending.text;
      await runAnalyze();
    }
  });

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
      watchJob(apiURL(job.eventsUrl));
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
      return 'Could not reach the authenticated API. Sign in again, then retry the shared URL.';
    }
    return api?.error?.message ?? 'Something went wrong.';
  }
</script>

<svelte:head>
  <title>AlaTube</title>
</svelte:head>

<main class="shell">
  <section class="panel">
    <div>
      <p class="eyebrow">Private media analysis</p>
      <h1>AlaTube</h1>
    </div>

    <form on:submit|preventDefault={runAnalyze} class="analyze-form">
      <label for="url">YouTube URL</label>
      <div class="input-row">
        <input id="url" bind:value={url} autocomplete="off" placeholder="Paste or share a YouTube URL" />
        <button disabled={busy || !url.trim()}>{busy ? 'Working' : 'Analyze'}</button>
      </div>
    </form>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    {#if analysis}
      <article class="result">
        {#if analysis.thumbnailUrl}
          <img src={analysis.thumbnailUrl} alt="" />
        {/if}
        <div>
          <h2>{analysis.title || analysis.videoId}</h2>
          <p>{analysis.durationSeconds ? Math.round(analysis.durationSeconds / 60) + ' min' : 'Duration unavailable'}</p>
        </div>
      </article>

      <div class="selectors">
        <label>
          Video
          <select bind:value={selectedVideo}>
            {#each videoFormats as format}
              <option value={format.formatId}>
                {format.height ? `${format.height}p` : format.formatId} · {format.container ?? 'media'} · {readableBytes(format.estimatedBytes)}
              </option>
            {/each}
          </select>
        </label>

        <label>
          Audio
          <select bind:value={selectedAudio}>
            <option value="">Use muxed audio if available</option>
            {#each audioFormats as format}
              <option value={format.formatId}>{format.container ?? format.formatId} · {readableBytes(format.estimatedBytes)}</option>
            {/each}
          </select>
        </label>
      </div>

      <button class="primary" on:click={runJob} disabled={busy || !selectedVideo}>Create download</button>
    {/if}

    {#if jobEvent}
      <section class="status">
        <div class="meter"><span style={`width: ${Math.max(8, jobEvent.progress * 100)}%`}></span></div>
        <p>{jobEvent.message}</p>
        {#if jobEvent.downloadUrl}
          <a class="download" href={apiURL(jobEvent.downloadUrl)}>Download file</a>
        {/if}
        {#if jobEvent.error}
          <p class="error">{jobEvent.error.message}</p>
        {/if}
      </section>
    {/if}
  </section>
</main>

<style>
  :global(body) {
    margin: 0;
    font-family:
      Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    color: #172124;
    background: #f6f2ea;
  }

  .shell {
    min-height: 100vh;
    display: grid;
    place-items: center;
    padding: 24px;
  }

  .panel {
    width: min(760px, 100%);
    display: grid;
    gap: 22px;
  }

  .eyebrow {
    margin: 0 0 6px;
    color: #0b5c6b;
    font-size: 0.86rem;
    font-weight: 700;
    text-transform: uppercase;
  }

  h1,
  h2,
  p {
    margin-top: 0;
  }

  h1 {
    margin-bottom: 0;
    font-size: 3rem;
  }

  h2 {
    margin-bottom: 6px;
    font-size: 1.1rem;
  }

  label {
    display: grid;
    gap: 8px;
    font-weight: 700;
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
  select,
  button,
  .download {
    min-height: 44px;
    border-radius: 8px;
    font: inherit;
  }

  input,
  select {
    border: 1px solid #c9d2d1;
    padding: 0 12px;
    background: white;
  }

  button,
  .download {
    border: 0;
    padding: 0 18px;
    background: #0b5c6b;
    color: white;
    font-weight: 800;
    cursor: pointer;
    text-decoration: none;
    display: inline-grid;
    place-items: center;
  }

  button:disabled {
    opacity: 0.55;
    cursor: wait;
  }

  .primary,
  .download {
    width: fit-content;
  }

  .error {
    color: #9f1d1d;
    font-weight: 700;
  }

  .result {
    display: grid;
    grid-template-columns: 144px 1fr;
    gap: 16px;
    align-items: center;
    padding: 14px;
    border: 1px solid #d8dedc;
    background: white;
    border-radius: 8px;
  }

  .result img {
    width: 144px;
    aspect-ratio: 16 / 9;
    object-fit: cover;
    border-radius: 6px;
  }

  .meter {
    height: 10px;
    overflow: hidden;
    border-radius: 999px;
    background: #d9e4e2;
  }

  .meter span {
    display: block;
    height: 100%;
    background: #168390;
  }

  @media (max-width: 620px) {
    .shell {
      align-items: start;
      padding: 18px;
    }

    h1 {
      font-size: 2.25rem;
    }

    .input-row,
    .result {
      grid-template-columns: 1fr;
    }

    .result img,
    button,
    .download {
      width: 100%;
    }
  }
</style>
