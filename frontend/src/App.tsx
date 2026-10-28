import { type FormEvent, type ReactNode, useCallback, useEffect, useMemo, useState } from 'react';
import {
  Activity, ArrowDownRight, ArrowUpRight, Bell, Check, CheckCircle2,
  ChevronDown, ChevronRight, CircleAlert, Clock3, Code2, Database,
  FileCheck2, FileJson, Gauge, GitCompareArrows, History, Layers3,
  LoaderCircle, Plus, RefreshCw, Search, Settings2, ShieldCheck, Upload,
  X, Zap,
} from 'lucide-react';

type Rule = {
  source_field: string;
  destination_field: string;
  required: boolean;
  normalization: string;
};
type Config = {
  name: string;
  rules: Rule[];
  match_key?: { source_field: string; destination_field: string };
};
type Integration = {
  id: string;
  name: string;
  config: Config;
  created_at: string;
  updated_at: string;
};
type Issue = {
  code?: string;
  field?: string;
  message?: string;
  record_key?: string;
  details?: Array<{ field?: string; expected?: unknown; actual?: unknown }>;
};
type ValidationResult = {
  valid?: boolean;
  checked?: number;
  source_records?: number;
  destination_records?: number;
  matched_records?: number;
  passed_records?: number;
  failed_records?: number;
  issues?: Issue[];
  [key: string]: unknown;
};
type Report = {
  integration_id?: string;
  integration_name?: string;
  valid: boolean;
  result: ValidationResult;
};
type Run = {
  id: string;
  integration_id: string;
  integration_name: string;
  status: string;
  report: any;
  created_at: string;
};

const API = '/api/v1';
const sampleSource = JSON.stringify([
  { employee_id: '00427', email: ' USER@EXAMPLE.COM ', department: 'FIN' },
  { employee_id: '00428', email: 'maria@example.com', department: 'OPS' },
  { employee_id: '00429', email: 'alex@example.com', department: 'HR' },
], null, 2);
const sampleDestination = JSON.stringify([
  { staffCode: '00427', emailAddress: 'user@example.com', departmentCode: 'fin' },
  { staffCode: '00428', emailAddress: 'maria@example.com', departmentCode: 'ops' },
  { staffCode: '00429', emailAddress: 'alex@example.com', departmentCode: 'hr' },
], null, 2);

class ApiError extends Error {
  constructor(message: string, public status: number, public data: any) {
    super(message);
    this.name = 'ApiError';
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
  });
  const text = await response.text();
  let data: any = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = text; }
  if (!response.ok) {
    throw new ApiError(
      data?.error || data?.message || `Request failed (${response.status})`,
      response.status,
      data,
    );
  }
  return data as T;
}

function date(value?: string) {
  if (!value) return '—';
  const d = new Date(value);
  return Number.isNaN(d.getTime())
    ? value
    : d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}
function shortId(id: string) { return id ? `${id.slice(0, 8)}…${id.slice(-4)}` : '—'; }
function resultFromRun(run: Run): ValidationResult {
  return run.report?.result ?? run.report?.report?.result ?? {};
}
function reportFromResponse(data: any): Report | null {
  const candidate = data?.report ?? data?.run?.report ?? data?.run?.report?.report;
  if (!candidate) return null;
  if (typeof candidate.valid === 'boolean' && candidate.result) return candidate as Report;
  if (candidate.report && typeof candidate.report.valid === 'boolean') return candidate.report as Report;
  return null;
}

export default function App() {
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [active, setActive] = useState('overview');
  const [selectedId, setSelectedId] = useState('');
  const [loading, setLoading] = useState(true);
  const [apiConnected, setApiConnected] = useState(false);
  const [apiError, setApiError] = useState('');
  const [search, setSearch] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [integrationName, setIntegrationName] = useState('New integration');
  const [matchSource, setMatchSource] = useState('id');
  const [matchDestination, setMatchDestination] = useState('id');
  const [rulesText, setRulesText] = useState(JSON.stringify([
    { source_field: 'id', destination_field: 'id', required: true, normalization: 'none' },
  ], null, 2));
  const [sourceText, setSourceText] = useState(sampleSource);
  const [destinationText, setDestinationText] = useState(sampleDestination);
  const [running, setRunning] = useState(false);
  const [report, setReport] = useState<Report | null>(null);
  const [runError, setRunError] = useState('');
  const [selectedRun, setSelectedRun] = useState<Run | null>(null);
  const [toast, setToast] = useState('');

  const refresh = useCallback(async () => {
    setLoading(true);
    setApiError('');
    try {
      const [i, r] = await Promise.all([
        request<{ items: Integration[] }>(`${API}/integrations`),
        request<{ items: Run[] }>(`${API}/runs?limit=100`),
      ]);
      const nextIntegrations = i.items || [];
      setIntegrations(nextIntegrations);
      setRuns(r.items || []);
      setApiConnected(true);
      setSelectedId(current =>
        current && nextIntegrations.some(item => item.id === current)
          ? current
          : nextIntegrations[0]?.id || '',
      );
    } catch (error) {
      setApiConnected(false);
      setApiError(error instanceof Error ? error.message : 'Unable to connect to the API');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(''), 3200);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const selectedIntegration = integrations.find(item => item.id === selectedId) || integrations[0];
  const passed = runs.filter(run => run.status === 'passed').length;
  const failed = runs.filter(run => run.status === 'failed').length;
  const passRate = runs.length ? Math.round((passed / runs.length) * 100) : 0;
  const filteredIntegrations = useMemo(
    () => integrations.filter(item => `${item.name} ${item.id}`.toLowerCase().includes(search.toLowerCase())),
    [integrations, search],
  );
  const recentRuns = runs.slice(0, 6);

  function resetCreateForm() {
    setIntegrationName('New integration');
    setMatchSource('id');
    setMatchDestination('id');
    setRulesText(JSON.stringify([
      { source_field: 'id', destination_field: 'id', required: true, normalization: 'none' },
    ], null, 2));
  }

  async function createIntegration(event: FormEvent) {
    event.preventDefault();
    setCreating(true);
    setApiError('');
    try {
      const rules = JSON.parse(rulesText) as Rule[];
      if (!Array.isArray(rules) || rules.length === 0) {
        throw new Error('Field rules must be a non-empty JSON array.');
      }
      const config: Config = {
        name: integrationName.trim(),
        rules,
        ...(matchSource.trim() && matchDestination.trim()
          ? { match_key: { source_field: matchSource.trim(), destination_field: matchDestination.trim() } }
          : {}),
      };
      const created = await request<Integration>(`${API}/integrations`, {
        method: 'POST',
        body: JSON.stringify(config),
      });
      await refresh();
      setSelectedId(created.id);
      setShowCreate(false);
      setActive('integrations');
      setToast('Integration created successfully');
    } catch (error) {
      setApiError(error instanceof Error ? error.message : 'Could not create integration');
    } finally {
      setCreating(false);
    }
  }

  async function runValidation(event: FormEvent) {
    event.preventDefault();
    if (!selectedIntegration) {
      setRunError('Create an integration first.');
      return;
    }
    setRunning(true);
    setRunError('');
    setReport(null);
    setSelectedRun(null);
    try {
      const source = JSON.parse(sourceText);
      const destination = JSON.parse(destinationText);
      const data = await request<{ report: Report; run: Run }>(
        `${API}/integrations/${selectedIntegration.id}/runs`,
        { method: 'POST', body: JSON.stringify({ source, destination }) },
      );
      const nextReport = reportFromResponse(data) || data.report;
      setReport(nextReport);
      setToast(nextReport?.valid ? 'Validation passed' : 'Validation completed with issues');
      await refresh();
    } catch (error) {
      // The API uses HTTP 422 for a completed validation that found data-quality issues.
      if (error instanceof ApiError && error.status === 422) {
        const failedReport = reportFromResponse(error.data);
        if (failedReport) {
          setReport(failedReport);
          setToast('Validation completed with issues');
          await refresh();
        } else {
          setRunError(
            `${error.message}. The API returned 422 but did not include a report in the expected response shape.`,
          );
          await refresh();
        }
      } else {
        setRunError(
          error instanceof SyntaxError
            ? `Invalid JSON: ${error.message}`
            : error instanceof Error
              ? error.message
              : 'Validation failed',
        );
      }
    } finally {
      setRunning(false);
    }
  }

  async function openRun(run: Run) {
    setSelectedRun(run);
    setActive('history');
    try {
      const detail = await request<Run>(`${API}/runs/${run.id}`);
      setSelectedRun(detail);
    } catch {
      // Keep the run list's copy if the detail endpoint is unavailable.
    }
  }

  function loadSample() {
    setSourceText(sampleSource);
    setDestinationText(sampleDestination);
    setRunError('');
    setReport(null);
  }
  function loadFailureSample() {
    setSourceText(sampleSource);
    setDestinationText(JSON.stringify([
      { staffCode: '00427', emailAddress: 'user@example.com', departmentCode: 'sales' },
      { staffCode: '00999', emailAddress: 'extra@example.com', departmentCode: 'ops' },
    ], null, 2));
    setRunError('');
    setReport(null);
  }
  function loadExampleConfig() {
    setIntegrationName('HR-to-Payroll Integration');
    setMatchSource('employee_id');
    setMatchDestination('staffCode');
    setRulesText(JSON.stringify([
      { source_field: 'employee_id', destination_field: 'staffCode', required: true, normalization: 'none' },
      { source_field: 'email', destination_field: 'emailAddress', required: true, normalization: 'trim_lowercase' },
      { source_field: 'department', destination_field: 'departmentCode', required: true, normalization: 'trim_lowercase' },
    ], null, 2));
  }

  const nav = [
    { id: 'overview', label: 'Overview', icon: Gauge },
    { id: 'integrations', label: 'Integrations', icon: Layers3, count: integrations.length },
    { id: 'validate', label: 'Run validation', icon: GitCompareArrows },
    { id: 'history', label: 'Run history', icon: History },
  ];

  return <div className="app-shell">
    <aside className="sidebar">
      <div className="brand"><div className="brand-mark"><ShieldCheck size={22} /></div><div><strong>integration<span>guard</span></strong><small>DATA QUALITY PLATFORM</small></div></div>
      <div className="workspace"><div className="workspace-icon">IG</div><div className="workspace-copy"><b>Engineering workspace</b><span>Local environment</span></div><ChevronDown size={15} className="muted" /></div>
      <div className="nav-label">WORKSPACE</div>
      <nav>{nav.map(item => <button key={item.id} className={`nav-item ${active === item.id ? 'active' : ''}`} onClick={() => setActive(item.id)}><item.icon size={18} /><span>{item.label}</span>{item.count !== undefined && <em>{item.count}</em>}</button>)}</nav>
      <div className="sidebar-bottom">
        <div className={`side-status ${apiConnected ? 'connected' : 'disconnected'}`}><span className="pulse-dot" /><div><b>{apiConnected ? 'API connected' : 'API disconnected'}</b><small>{apiConnected ? 'localhost · port 8080' : 'Connection unavailable'}</small></div>{apiConnected ? <Check size={15} /> : <CircleAlert size={15} />}</div>
        <button className="nav-item" onClick={() => void refresh()}><RefreshCw size={17} /><span>Refresh data</span></button>
        <div className="sidebar-user"><div className="avatar">IG</div><div><b>Integration Guard</b><small>Developer edition</small></div><Settings2 size={17} className="muted" /></div>
      </div>
    </aside>

    <main className="main-area">
      <header className="topbar"><div className="breadcrumbs"><span>Workspace</span><ChevronRight size={14} /><b>{nav.find(item => item.id === active)?.label || 'Overview'}</b></div><div className="top-actions"><div className="env-pill"><span className="pulse-dot" /> LOCAL ENVIRONMENT</div><button className="icon-button" title="Refresh" onClick={() => void refresh()}><RefreshCw size={17} /></button><button className="icon-button" title="Notifications"><Bell size={17} /><i /></button><div className="avatar avatar-small">IG</div></div></header>
      <div className="page-content">
        {apiError && <div className="alert alert-error"><CircleAlert size={18} /><div><b>{active === 'integrations' && showCreate ? 'Could not create integration' : 'API request failed'}</b><p>{apiError}. Check that the API container is running, then refresh.</p></div><button onClick={() => void refresh()} className="alert-action">Retry</button></div>}

        {active === 'overview' && <>
          <section className="welcome-row"><div><div className="eyebrow"><span className="eyebrow-line" /> INTEGRATION OBSERVABILITY</div><h1>Integration overview</h1><p className="subtitle">Monitor data quality and catch issues before they reach downstream systems.</p></div><button className="primary-button" onClick={() => setActive('validate')}><Zap size={16} /> Run validation</button></section>
          <section className="metric-grid">
            <Metric label="Total integrations" value={integrations.length} icon={<Layers3 size={18} />} detail="Configured pipelines" tone="blue" />
            <Metric label="Validation runs" value={runs.length} icon={<Activity size={18} />} detail="Saved run history" tone="violet" />
            <Metric label="Pass rate" value={`${passRate}%`} icon={<CheckCircle2 size={18} />} detail={`${passed} successful runs`} tone="green" />
            <Metric label="Failed runs" value={failed} icon={<CircleAlert size={18} />} detail="Require investigation" tone="orange" />
          </section>
          <section className="overview-grid"><div className="panel pipeline-panel"><div className="panel-heading"><div><h2>Integration health</h2><p>Configured pipelines and their latest known activity</p></div><button className="text-button" onClick={() => setActive('integrations')}>View all <ChevronRight size={15} /></button></div>
            {loading ? <Loading /> : integrations.length === 0 ? <Empty icon={<Layers3 />} title="No integrations yet" description="Create your first integration to start comparing data between systems." action={<button className="primary-button" onClick={() => setShowCreate(true)}><Plus size={16} /> Create integration</button>} /> : <div className="table-wrap"><table><thead><tr><th>INTEGRATION</th><th>RULES</th><th>LATEST RUN</th><th>STATUS</th><th /></tr></thead><tbody>{integrations.slice(0, 5).map(item => { const last = runs.find(run => run.integration_id === item.id); return <tr key={item.id} onClick={() => { setSelectedId(item.id); setActive('integrations'); }}><td><div className="integration-cell"><div className="integration-icon"><GitCompareArrows size={17} /></div><div><b>{item.name}</b><small>{shortId(item.id)}</small></div></div></td><td><span className="rule-count">{item.config?.rules?.length || 0} rules</span></td><td className="date-cell">{date(last?.created_at)}</td><td><Status status={last?.status || 'not_run'} /></td><td><ChevronRight size={16} className="muted" /></td></tr>; })}</tbody></table></div>}
          </div><div className="panel activity-panel"><div className="panel-heading"><div><h2>Recent activity</h2><p>Latest validation events</p></div><div className="heading-icon"><Activity size={17} /></div></div>
            {recentRuns.length === 0 ? <Empty icon={<Clock3 />} title="No activity yet" description="Your validation runs will appear here." /> : <div className="activity-list">{recentRuns.slice(0, 5).map(run => <button className="activity-item" key={run.id} onClick={() => void openRun(run)}><span className={`activity-symbol ${run.status === 'passed' ? 'ok' : 'bad'}`}>{run.status === 'passed' ? <Check size={15} /> : <CircleAlert size={15} />}</span><span className="activity-copy"><b>{run.integration_name}</b><small>{run.status === 'passed' ? 'Validation passed' : run.status === 'failed' ? 'Validation failed' : `Status: ${run.status}`} · {date(run.created_at)}</small></span><ChevronRight size={15} className="muted" /></button>)}</div>}
            <button className="activity-footer" onClick={() => setActive('history')}>Open run history <ArrowUpRight size={15} /></button>
          </div></section>
          <section className="bottom-banner"><div className="banner-icon"><ShieldCheck size={22} /></div><div><b>Validate with confidence</b><p>Compare records, normalize values, and surface mismatches with traceable run reports.</p></div><button className="secondary-button" onClick={() => setActive('integrations')}>Manage integrations <ArrowUpRight size={15} /></button></section>
        </>}

        {active === 'integrations' && <>
          <section className="welcome-row"><div><div className="eyebrow"><span className="eyebrow-line" /> PIPELINE CONFIGURATION</div><h1>Integrations</h1><p className="subtitle">Define field mappings and record matching rules for every data pipeline.</p></div><button className="primary-button" onClick={() => { resetCreateForm(); setShowCreate(true); }}><Plus size={16} /> New integration</button></section>
          <div className="panel integrations-panel"><div className="list-toolbar"><div className="search-box"><Search size={17} /><input placeholder="Search integrations…" value={search} onChange={event => setSearch(event.target.value)} /></div><span className="results-count">{filteredIntegrations.length} integration{filteredIntegrations.length === 1 ? '' : 's'}</span></div>
            {loading ? <Loading /> : filteredIntegrations.length === 0 ? <Empty icon={<Layers3 />} title={search ? 'No matching integrations' : 'Your workspace is empty'} description={search ? 'Try another search term.' : 'Create a configuration to define how source and destination records should match.'} action={!search ? <button className="primary-button" onClick={() => { resetCreateForm(); setShowCreate(true); }}><Plus size={16} /> Create integration</button> : undefined} /> : <div className="integration-cards">{filteredIntegrations.map(item => { const last = runs.find(run => run.integration_id === item.id); return <article className={`integration-card ${selectedId === item.id ? 'selected' : ''}`} key={item.id} onClick={() => setSelectedId(item.id)}><div className="card-top"><div className="integration-icon large"><GitCompareArrows size={19} /></div><Status status={last?.status || 'not_run'} /></div><h3>{item.name}</h3><p className="card-id">ID · {shortId(item.id)}</p><div className="mapping-preview"><span>Source key</span><code>{item.config?.match_key?.source_field || 'single record'}</code><ArrowDownRight size={14} /><span>Destination key</span><code>{item.config?.match_key?.destination_field || 'single record'}</code></div><div className="card-bottom"><span><Code2 size={14} /> {item.config?.rules?.length || 0} field rules</span><span>Updated {date(item.updated_at)}</span></div><div className="card-actions"><button className="secondary-button small" onClick={event => { event.stopPropagation(); setSelectedId(item.id); setActive('validate'); }}>Run validation <ArrowUpRight size={14} /></button><button className="text-button" onClick={event => { event.stopPropagation(); setSelectedId(item.id); setActive('validate'); }}>Inspect <ChevronRight size={15} /></button></div></article>; })}</div>}
          </div>
          {selectedIntegration && <div className="panel detail-strip"><div className="detail-title"><div className="integration-icon"><FileCheck2 size={18} /></div><div><b>{selectedIntegration.name}</b><small>Selected integration · {shortId(selectedIntegration.id)}</small></div></div><div className="detail-stats"><span><b>{selectedIntegration.config?.rules?.length || 0}</b> field rules</span><span><b>{runs.filter(run => run.integration_id === selectedIntegration.id).length}</b> runs</span></div><button className="primary-button" onClick={() => setActive('validate')}>Validate data <ChevronRight size={16} /></button></div>}
        </>}

        {active === 'validate' && <>
          <section className="welcome-row"><div><div className="eyebrow"><span className="eyebrow-line" /> DATA RECONCILIATION</div><h1>Run validation</h1><p className="subtitle">Compare source and destination payloads against a saved integration's rules.</p></div><button className="secondary-button" onClick={loadSample}><RefreshCw size={15} /> Load passing sample</button></section>
          <div className="panel validation-config"><div className="field-label"><span>INTEGRATION CONFIGURATION</span><span className="required-label">Required</span></div><div className="select-wrap"><Layers3 size={17} /><select value={selectedIntegration?.id || ''} onChange={event => { setSelectedId(event.target.value); setReport(null); setRunError(''); }} disabled={!integrations.length}><option value="" disabled>Select an integration</option>{integrations.map(item => <option value={item.id} key={item.id}>{item.name}</option>)}</select><ChevronDown size={15} /></div>{selectedIntegration && <div className="config-summary"><span><Database size={14} /> Match key: <code>{selectedIntegration.config?.match_key?.source_field || 'single record'}</code> → <code>{selectedIntegration.config?.match_key?.destination_field || 'single record'}</code></span><span><Code2 size={14} /> {selectedIntegration.config?.rules?.length || 0} field rules</span></div>}</div>
          <form onSubmit={runValidation}><div className="payload-grid"><PayloadEditor title="Source payload" description="Data from the upstream system" value={sourceText} onChange={setSourceText} icon={<Upload size={17} />} onFile={file => { void file.text().then(setSourceText).catch(() => setRunError('Could not read the selected file.')); }} /><PayloadEditor title="Destination payload" description="Data from the downstream system" value={destinationText} onChange={setDestinationText} icon={<Database size={17} />} onFile={file => { void file.text().then(setDestinationText).catch(() => setRunError('Could not read the selected file.')); }} /></div>
            <div className="validation-actions"><button type="button" className="text-button" onClick={loadFailureSample}>Load failure scenario <ArrowUpRight size={14} /></button><div><span className="editor-hint"><FileJson size={14} /> JSON object or array</span><button className="primary-button" type="submit" disabled={running || !selectedIntegration}>{running ? <LoaderCircle size={16} className="spin" /> : <GitCompareArrows size={16} />} {running ? 'Validating…' : 'Run validation'}</button></div></div>
          </form>
          {runError && <div className="alert alert-error"><CircleAlert size={18} /><div><b>Validation could not be completed</b><p>{runError}</p></div></div>}
          {report && <ReportPanel report={report} onHistory={() => setActive('history')} />}
          {!report && !runError && <div className="validation-tip"><div className="tip-icon"><ShieldCheck size={19} /></div><div><b>What gets checked?</b><p>Record matching, required fields, normalization rules, and value consistency. Every run is saved for later inspection.</p></div></div>}
        </>}

        {active === 'history' && <>
          <section className="welcome-row"><div><div className="eyebrow"><span className="eyebrow-line" /> AUDIT TRAIL</div><h1>Run history</h1><p className="subtitle">Every validation run, with its outcome and diagnostic report.</p></div><button className="secondary-button" onClick={() => void refresh()}><RefreshCw size={15} /> Refresh history</button></section>
          <div className="panel history-panel"><div className="panel-heading"><div><h2>Validation runs</h2><p>{runs.length} saved run{runs.length === 1 ? '' : 's'} · newest first</p></div><div className="heading-icon"><History size={17} /></div></div>
            {loading ? <Loading /> : runs.length === 0 ? <Empty icon={<History />} title="No runs recorded" description="Run a validation to create your first audit record." action={<button className="primary-button" onClick={() => setActive('validate')}>Run validation</button>} /> : <div className="table-wrap"><table><thead><tr><th>RUN ID</th><th>INTEGRATION</th><th>RESULT</th><th>RECORDS</th><th>CREATED</th><th /></tr></thead><tbody>{runs.map(run => { const result = resultFromRun(run); return <tr key={run.id} className={selectedRun?.id === run.id ? 'row-selected' : ''} onClick={() => void openRun(run)}><td><code className="run-id">{shortId(run.id)}</code></td><td><b>{run.integration_name}</b></td><td><Status status={run.status} /></td><td><span className="record-summary">{result.passed_records ?? 0} passed <i>·</i> {result.failed_records ?? 0} failed</span></td><td className="date-cell">{date(run.created_at)}</td><td><ChevronRight size={16} className="muted" /></td></tr>; })}</tbody></table></div>}
          </div>
          {selectedRun && <RunDetail run={selectedRun} onClose={() => setSelectedRun(null)} />}
        </>}

        <footer className="footer"><span>INTEGRATION GUARD <b>v0.1</b></span><span><span className="pulse-dot" /> {apiConnected ? 'API-backed local workspace' : 'API connection unavailable'}</span><span>Built for traceable data quality</span></footer>
      </div>
    </main>

    {showCreate && <div className="modal-backdrop" onMouseDown={event => { if (event.target === event.currentTarget) setShowCreate(false); }}><form className="modal" onSubmit={createIntegration}><div className="modal-heading"><div><div className="eyebrow">NEW PIPELINE</div><h2>Create integration</h2><p>Define the match key and field-level validation rules.</p></div><button type="button" className="icon-button" onClick={() => setShowCreate(false)}><X size={18} /></button></div>
      <button type="button" className="text-button" onClick={loadExampleConfig}>Load HR-to-Payroll example</button>
      {apiError && <div className="alert alert-error"><CircleAlert size={16} /><p>{apiError}</p></div>}
      <label className="form-field"><span>Integration name</span><input value={integrationName} onChange={event => setIntegrationName(event.target.value)} required minLength={2} placeholder="e.g. HR to Payroll" /></label>
      <div className="form-two"><label className="form-field"><span>Source match key</span><input value={matchSource} onChange={event => setMatchSource(event.target.value)} placeholder="employee_id" /></label><label className="form-field"><span>Destination match key</span><input value={matchDestination} onChange={event => setMatchDestination(event.target.value)} placeholder="staffCode" /></label></div>
      <label className="form-field"><span>Field rules <small>JSON</small></span><textarea className="rules-input" value={rulesText} onChange={event => setRulesText(event.target.value)} spellCheck={false} required /></label>
      <div className="modal-note"><CircleAlert size={15} /><span>Each rule needs <code>source_field</code>, <code>destination_field</code>, <code>required</code>, and <code>normalization</code>. Normalization: <code>none</code> or <code>trim_lowercase</code>.</span></div>
      <div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setShowCreate(false)}>Cancel</button><button className="primary-button" disabled={creating}>{creating ? <LoaderCircle size={16} className="spin" /> : <Plus size={16} />} Create integration</button></div>
    </form></div>}
    {toast && <div className="toast"><CheckCircle2 size={17} />{toast}</div>}
  </div>;
}

function Metric({ label, value, icon, detail, tone }: { label: string; value: string | number; icon: ReactNode; detail: string; tone: string }) {
  return <div className="metric-card"><div className={`metric-icon ${tone}`}>{icon}</div><div className="metric-label">{label}</div><div className="metric-value">{value}</div><div className="metric-detail">{detail}</div></div>;
}
function Status({ status }: { status: string }) {
  const pass = status === 'passed';
  const fail = status === 'failed';
  return <span className={`status ${pass ? 'status-pass' : fail ? 'status-fail' : 'status-idle'}`}><span />{pass ? 'Passed' : fail ? 'Failed' : status === 'not_run' ? 'Not run' : status}</span>;
}
function Empty({ icon, title, description, action }: { icon: ReactNode; title: string; description: string; action?: ReactNode }) {
  return <div className="empty-state"><div className="empty-icon">{icon}</div><b>{title}</b><p>{description}</p>{action}</div>;
}
function Loading() { return <div className="loading-state"><LoaderCircle className="spin" size={20} /> Loading workspace data…</div>; }

function PayloadEditor({ title, description, value, onChange, icon, onFile }: {
  title: string; description: string; value: string; onChange: (value: string) => void;
  icon: ReactNode; onFile: (file: File) => void;
}) {
  const [error, setError] = useState('');
  function check() {
    try { JSON.parse(value); setError(''); }
    catch (caught) { setError(caught instanceof Error ? caught.message : 'Invalid JSON'); }
  }
  return <div className="panel payload-panel"><div className="payload-heading"><div className="payload-title-icon">{icon}</div><div><h2>{title}</h2><p>{description}</p></div><label className="upload-button" title="Upload JSON file"><Upload size={15} /><span>Upload</span><input type="file" accept="application/json,.json" onChange={event => { const file = event.target.files?.[0]; if (file) onFile(file); event.currentTarget.value = ''; }} /></label></div><div className="editor-toolbar"><span><span className="editor-dot" /> JSON</span><button type="button" onClick={check}>Format check</button></div><textarea className="json-editor" spellCheck={false} value={value} onChange={event => { onChange(event.target.value); setError(''); }} aria-label={title} />{error && <div className="json-error"><CircleAlert size={13} />{error}</div>}<div className="editor-foot"><span>{value.split('\n').length} lines</span><span>UTF-8 · JSON</span></div></div>;
}

function ReportPanel({ report, onHistory }: { report: Report; onHistory: () => void }) {
  const result = report.result || {};
  const issues = result.issues || [];
  return <section className="panel report-panel"><div className="report-header"><div className={`report-icon ${report.valid ? 'success' : 'failure'}`}>{report.valid ? <CheckCircle2 size={23} /> : <CircleAlert size={23} />}</div><div><div className="eyebrow">VALIDATION REPORT</div><h2>{report.valid ? 'All checks passed' : 'Validation issues detected'}</h2><p>{report.integration_name || 'Integration validation'}</p></div><span className={`report-status ${report.valid ? 'success' : 'failure'}`}>{report.valid ? 'PASS' : 'FAIL'}</span></div><div className="report-metrics"><div><span>Source records</span><b>{result.source_records ?? result.checked ?? '—'}</b></div><div><span>Destination records</span><b>{result.destination_records ?? '—'}</b></div><div><span>Matched records</span><b>{result.matched_records ?? '—'}</b></div><div><span>Passed records</span><b className="green-text">{result.passed_records ?? (report.valid ? result.checked : '—')}</b></div><div><span>Failed records</span><b className="red-text">{result.failed_records ?? issues.length}</b></div></div>{issues.length > 0 ? <div className="issues-list"><div className="issues-title"><b>Issues</b><span>{issues.length} found</span></div>{issues.map((issue, index) => <div className="issue-row" key={`${issue.code || 'issue'}-${index}`}><CircleAlert size={16} /><div><b>{issue.record_key ? `Record ${issue.record_key} · ` : ''}{issue.code || issue.field || 'Validation issue'}</b><p>{issue.message || 'No issue description provided.'}</p>{issue.details?.map((detail, detailIndex) => <small key={detailIndex}>{detail.field || 'Field'}: expected <code>{JSON.stringify(detail.expected)}</code>, received <code>{JSON.stringify(detail.actual)}</code></small>)}</div></div>)}</div> : <div className="success-note"><CheckCircle2 size={17} /> No issues found. All records satisfy the configured validation rules.</div>}<div className="report-footer"><span><Database size={14} /> Run saved to history</span><button className="text-button" onClick={onHistory}>View run history <ArrowUpRight size={14} /></button></div></section>;
}

function RunDetail({ run, onClose }: { run: Run; onClose: () => void }) {
  const result = resultFromRun(run);
  const issues: Issue[] = result.issues || [];
  return <section className="panel run-detail"><div className="panel-heading"><div><div className="eyebrow">RUN DETAILS</div><h2>{run.integration_name}</h2><p>{run.id} · {date(run.created_at)}</p></div><button className="icon-button" onClick={onClose}><X size={17} /></button></div><div className="detail-run-summary"><Status status={run.status} /><span>{result.source_records ?? '—'} source records</span><span>{result.destination_records ?? '—'} destination records</span><span>{result.passed_records ?? 0} passed</span><span>{result.failed_records ?? 0} failed</span></div>{issues.length ? <div className="issues-list">{issues.map((issue, index) => <div className="issue-row" key={`${issue.code || 'issue'}-${index}`}><CircleAlert size={16} /><div><b>{issue.record_key ? `Record ${issue.record_key} · ` : ''}{issue.code || issue.field || 'Validation issue'}</b><p>{issue.message || 'No issue description provided.'}</p>{issue.details?.map((detail, detailIndex) => <small key={detailIndex}>{detail.field || 'Field'}: expected <code>{JSON.stringify(detail.expected)}</code>, received <code>{JSON.stringify(detail.actual)}</code></small>)}</div></div>)}</div> : <div className="success-note"><CheckCircle2 size={17} /> No validation issues recorded.</div>}<details className="raw-report"><summary>View raw report JSON</summary><pre>{JSON.stringify(run.report, null, 2)}</pre></details></section>;
}
