<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>流量分析 · Xboard 管理后台</title>
  <style>
    :root{--ink:#10213e;--muted:#8190a6;--blue:#1677ff;--bg:#f2f5f9;--line:#e7ecf2;--card:#fff;--hover:#f5f8fc;--shadow:0 2px 8px rgba(32,52,84,.06)}
    *{box-sizing:border-box}body{margin:0;color:var(--ink);background:var(--bg);font-family:Inter,"Microsoft YaHei",system-ui,-apple-system,sans-serif;font-size:14px}.layout{min-height:100vh;display:flex}.sidebar{width:248px;flex:0 0 248px;background:#fff;border-right:1px solid #e4e9ef;display:flex;flex-direction:column}.brand{height:74px;display:flex;align-items:center;padding:0 32px;border-bottom:1px solid #e8edf3;font-weight:780;font-size:20px;letter-spacing:-.03em}.brand-mark{height:28px;width:28px;border-radius:9px;background:linear-gradient(135deg,#1677ff,#52a4ff);margin-right:11px;display:grid;place-items:center;color:white;font-size:16px}.nav{padding:17px 11px;display:grid;gap:5px}.nav button{height:46px;border:0;background:transparent;color:#44536a;border-radius:8px;text-align:left;padding:0 13px;font:500 15px inherit;cursor:pointer;display:flex;align-items:center;gap:13px}.nav button:hover{background:#f5f8fc}.nav button.active{background:#edf5ff;color:#1677ff;font-weight:700}.nav-icon{width:22px;text-align:center;font-size:18px}.profile{margin-top:auto;padding:17px;border-top:1px solid #e9edf3;display:flex;align-items:center;gap:10px}.avatar{height:37px;width:37px;border-radius:50%;display:grid;place-items:center;background:#eaf3ff;color:#1677ff;font-weight:750;font-size:16px}.profile strong{display:block;font-size:14px}.profile small{color:var(--muted)}.content{min-width:0;flex:1;padding:33px 46px 48px;max-width:1800px;margin:auto}.header{display:flex;justify-content:space-between;gap:20px;align-items:start;margin-bottom:28px}.header h1{font-size:29px;line-height:1.1;letter-spacing:-.05em;margin:0 0 7px}.subtitle{color:#8190a5;font-size:15px}.controls{display:flex;gap:12px;align-items:center;flex-wrap:wrap}.segmented,.metric-toggle{border:1px solid #e1e7ee;border-radius:10px;background:#f5f7fa;padding:4px;display:flex;gap:2px}.segmented button,.metric-toggle button{border:0;background:transparent;border-radius:7px;color:#465671;padding:9px 14px;font:650 14px inherit;cursor:pointer;white-space:nowrap}.segmented button.active,.metric-toggle button.active{background:#fff;color:var(--blue);box-shadow:0 1px 5px rgba(32,62,104,.10)}.refresh{height:39px;border:1px solid #d9e1eb;border-radius:7px;background:#fff;color:#45546b;padding:0 14px;font:600 14px inherit;cursor:pointer}.refresh:hover{border-color:#a9b9cc}.notice{display:none;margin:0 0 18px;border:1px solid #f1d187;background:#fff9eb;color:#875b00;border-radius:9px;padding:11px 14px}.token-row{display:none;margin:0 0 18px;gap:8px}.token-row input{border:1px solid #dbe3ec;border-radius:7px;padding:9px 11px;width:310px;font:14px inherit}.token-row button{border:0;border-radius:7px;color:#fff;background:var(--blue);font-weight:700;padding:0 13px;cursor:pointer}.cards{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:18px;margin-bottom:23px}.card,.panel{background:var(--card);border:1px solid #e4e9ef;border-radius:13px;box-shadow:var(--shadow)}.card{padding:19px 21px}.card-label{font-size:14px;color:#76869e;font-weight:650}.card-value{font-size:25px;font-weight:790;letter-spacing:-.035em;margin-top:9px}.panel{overflow:hidden}.panel+.panel{margin-top:22px}.panel-head{min-height:82px;padding:0 23px;display:flex;align-items:center;justify-content:space-between;gap:16px;border-bottom:1px solid var(--line)}.panel-title{font-size:18px;font-weight:780}.tabs{display:flex;gap:8px;align-items:center}.tabs .metric-toggle{margin-left:4px}.stack{height:34px;margin:24px 25px 16px;background:#f0f3f6;border-radius:9px;overflow:hidden;display:flex}.stack span{height:100%;min-width:2px;background:#8e9290;border-right:2px solid white}.distribution{padding:0 23px 17px}.service-row{border-bottom:1px solid #e9edf2}.service-main{min-height:51px;display:grid;grid-template-columns:300px minmax(160px,1fr) 74px 205px 110px 22px;gap:14px;align-items:center;padding:0 7px;cursor:pointer}.service-main:hover{background:#fafcff}.site-name{display:flex;align-items:center;gap:12px;font-size:16px;min-width:0}.site-name span:last-child{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.favicon{display:inline-grid;place-items:center;width:21px;height:21px;flex:0 0 21px;border-radius:6px;color:#fff;font-size:12px;font-weight:800;line-height:1}.favicon.google{background:conic-gradient(from -45deg,#4285f4 0 25%,#34a853 0 50%,#fbbc05 0 75%,#ea4335 0)}.favicon.connectivity{border-radius:50%;background:linear-gradient(135deg,#9aa9c4,#536f9b)}.favicon.javdb{background:#d41480}.favicon.yandex{background:#fa554d}.favicon.cloudflare{background:#f48120}.favicon.microsoft{background:#4b8df8}.favicon.apple{background:#5e6772}.favicon.other{background:#9a9b99;border-radius:50%}.barline{height:6px;border-radius:99px;background:#edf1f5;overflow:hidden}.barline i{height:100%;display:block;background:#8e8e89;border-radius:inherit}.percent{font-size:15px;text-align:right;font-weight:780}.row-stats{white-space:nowrap;color:#7688a0;font-size:13px}.row-stats b{color:#6a7d96;font-weight:650}.chevron{color:#8090a6;transform:rotate(0deg);transition:transform .18s}.service-row.open .chevron{transform:rotate(180deg)}.children{display:none;margin:0 7px 10px;padding:5px 0 7px;border-radius:7px;background:#f5f7fa}.service-row.open .children{display:block}.child{min-height:36px;display:grid;grid-template-columns:300px minmax(160px,1fr) 74px 205px 110px 22px;gap:14px;align-items:center;padding:0 10px}.child .site-name{padding-left:28px;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:13px}.child .row-stats{text-align:right}.empty{padding:44px;text-align:center;color:#8795a9}.audit-head{display:flex;align-items:center;gap:9px}.status-dot{height:8px;width:8px;border-radius:50%;background:#25ba70}.audit-count{margin-left:auto;color:#71839d;font-size:13px;font-weight:500}.filters{display:flex;gap:10px;flex-wrap:wrap;padding:15px 23px;border-bottom:1px solid var(--line)}input,select{height:37px;border:1px solid #dce4ed;border-radius:7px;padding:0 10px;background:#fff;color:#33445e;font:14px inherit}.table-wrap{overflow:auto}.audit-table{width:100%;border-collapse:collapse;min-width:930px}.audit-table th,.audit-table td{padding:14px 20px;border-bottom:1px solid #e9edf2;text-align:left}.audit-table th{font-size:13px;color:#8291a7;font-weight:680}.audit-table tbody tr:hover{background:#fafcff}.protocol{display:inline-block;padding:4px 11px;border-radius:6px;background:#eaf3ff;color:#1677ff;font-weight:750;font-size:12px}.blocked{color:#df4d4d;font-weight:700}.view{display:none}.view.active{display:block}@media(max-width:1220px){.sidebar{width:210px;flex-basis:210px}.content{padding:28px 25px}.service-main,.child{grid-template-columns:230px minmax(130px,1fr) 68px 160px 85px 18px}}@media(max-width:920px){.sidebar{display:none}.cards{grid-template-columns:repeat(2,1fr)}.content{padding:22px 16px}.header{display:block}.controls{margin-top:16px}.service-main,.child{grid-template-columns:180px 1fr 65px 18px}.row-stats{display:none}.child .site-name{padding-left:10px}}@media(max-width:570px){.cards{grid-template-columns:1fr}.segmented{max-width:100%;overflow:auto}.service-main,.child{grid-template-columns:125px 1fr 54px 14px;gap:8px}.distribution{padding:0 10px 10px}.stack{margin:16px 12px}.panel-head{padding:0 14px}.tabs .metric-toggle{display:none}.service-main{padding:0}.child{padding:0 4px}}
  </style>
</head>
<body>
<div class="layout" data-api-base="{{ $apiBase }}">
  <aside class="sidebar"><div class="brand"><span class="brand-mark">A</span>Traffic Audit</div><nav class="nav"><button class="active" data-view="analysis"><span class="nav-icon">▥</span>流量分析</button><button data-view="audit"><span class="nav-icon">✎</span>流量审计</button></nav><div class="profile"><div class="avatar">A</div><div><strong>管理员</strong><small>Traffic Audit</small></div></div></aside>
  <main class="content">
    <div class="header"><div><h1 id="title">流量分析</h1><div class="subtitle" id="subtitle">节点流量去向——按服务与目标地址分组。</div></div><div class="controls"><div class="segmented" id="ranges"><button data-range="1h">最近 1 小时</button><button data-range="6h">最近 6 小时</button><button data-range="24h" class="active">最近 24 小时</button><button data-range="7d">最近 7 天</button><button data-range="14d">最近 14 天</button></div><button class="refresh" id="refresh">↻ 刷新</button></div></div>
    <div id="notice" class="notice"></div><div id="token-row" class="token-row"><input id="token" type="password" placeholder="粘贴 Xboard 管理员 API Token"><button id="save-token">连接后台</button></div>
    <section id="analysis" class="view active"><div class="cards"><div class="card"><div class="card-label">上传</div><div class="card-value" id="upload">—</div></div><div class="card"><div class="card-label">下载</div><div class="card-value" id="download">—</div></div><div class="card"><div class="card-label">总流量</div><div class="card-value" id="total">—</div></div><div class="card"><div class="card-label">连接数</div><div class="card-value" id="connections">—</div></div></div><section class="panel"><div class="panel-head"><span class="panel-title">流量分布</span><div class="tabs"><div class="metric-toggle"><button class="active" data-group="service">服务</button><button data-group="destination">目标地址</button></div><div class="metric-toggle"><button class="active" data-metric="traffic">流量</button><button data-metric="connections">连接数</button></div></div></div><div id="stack" class="stack"></div><div id="distribution" class="distribution"><div class="empty">暂无采集数据</div></div></section></section>
    <section id="audit" class="view"><section class="panel"><div class="panel-head"><div class="audit-head"><span class="status-dot"></span><span class="panel-title">实时流量审计记录</span></div><span id="event-count" class="audit-count">0 条记录</span></div><div class="filters"><input id="filter-destination" placeholder="目标地址"><input id="filter-rule" placeholder="规则标签"><select id="filter-action"><option value="">全部动作</option><option value="block">拦截</option><option value="observe">观察</option></select><button id="apply" class="refresh">筛选</button></div><div class="table-wrap"><table class="audit-table"><thead><tr><th>协议</th><th>来源</th><th>目标地址</th><th>动作 / 规则</th><th>时间</th></tr></thead><tbody id="events"><tr><td colspan="5" class="empty">正在加载…</td></tr></tbody></table></div></section></section>
  </main>
</div>
<script>
(()=>{const app=document.querySelector('.layout'),base=app.dataset.apiBase;let range='24h',group='service',metric='traffic';const saved='xboard-risk-audit-admin-token';const keys=['admin_token','adminToken','token','access_token','accessToken'];const token=()=>sessionStorage.getItem(saved)||localStorage.getItem(saved)||keys.map(k=>localStorage.getItem(k)).find(Boolean)||'';const bytes=n=>{n=Number(n||0);if(!n)return'0 B';const u=['B','KiB','MiB','GiB','TiB'];const i=Math.min(Math.floor(Math.log(n)/Math.log(1024)),u.length-1);return`${(n/1024**i).toFixed(i?2:0)} ${u[i]}`};const esc=s=>String(s??'—').replace(/[&<>'"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));const icon=n=>{const key=String(n||'Other').toLowerCase().replace(/\s+/g,'');const allowed=['google','connectivitycheck','javdb','yandex','cloudflare','microsoft','apple'];const cls=allowed.includes(key)?key:'other';const chars={google:'G',connectivitycheck:'◉',javdb:'番',yandex:'Y',cloudflare:'☁',microsoft:'⊞',apple:'●'};return`<span class="favicon ${cls}">${chars[key]||'•'}</span>`};const notice=(text,needToken)=>{const n=document.querySelector('#notice'),r=document.querySelector('#token-row');n.textContent=text;n.style.display='block';r.style.display=needToken?'flex':'none'};const req=async(path,params={})=>{const url=new URL(base+path,location.origin);Object.entries(params).forEach(([k,v])=>v!==''&&url.searchParams.set(k,v));const res=await fetch(url,{headers:token()?{Authorization:`Bearer ${token()}`}:{}});if(res.status===401||res.status===403){notice('审计数据仅面向 Xboard 管理员。未自动识别登录 Token 时，请粘贴管理员 API Token。',true);throw Error('auth')}if(!res.ok)throw Error(await res.text());return res.json()};const child=(x,max)=>`<div class="child"><div class="site-name">${icon('other')}<span>${esc(x.label)}</span></div><div class="barline"><i style="width:${Math.max(1,100*x.traffic_bytes/max)}%"></i></div><div class="percent">${(100*x.traffic_bytes/max).toFixed(1)}%</div><div class="row-stats">${bytes(x.traffic_bytes)}</div><div class="row-stats">${Number(x.connections).toLocaleString()} 个连接</div><span></span></div>`;const renderSummary=d=>{document.querySelector('#upload').textContent=bytes(d.cards.upload_bytes);document.querySelector('#download').textContent=bytes(d.cards.download_bytes);document.querySelector('#total').textContent=bytes(d.cards.total_bytes);document.querySelector('#connections').textContent=Number(d.cards.connections).toLocaleString();const items=d.distribution||[];const field=metric==='connections'?'connections':'traffic_bytes';const max=Math.max(...items.map(x=>Number(x[field])),1);document.querySelector('#stack').innerHTML=items.slice(0,16).map(x=>`<span style="width:${Math.max(1,100*x[field]/max)}%"></span>`).join('');document.querySelector('#distribution').innerHTML=items.length?items.map((x,i)=>`<div class="service-row ${i===0?'open':''}"><div class="service-main"><div class="site-name">${icon(x.label)}<span>${esc(x.label)}</span></div><div class="barline"><i style="width:${Math.max(1,100*x[field]/max)}%"></i></div><div class="percent">${(100*x[field]/max).toFixed(1)}%</div><div class="row-stats">↑ ${bytes(x.upload_bytes)}　↓ ${bytes(x.download_bytes)}</div><div class="row-stats">${Number(x.connections).toLocaleString()} 个连接</div><span class="chevron">⌄</span></div><div class="children">${(x.destinations||[]).map(c=>child(c,Math.max(x.traffic_bytes,1))).join('')}</div></div>`).join(''):'<div class="empty">该时间范围暂无采集数据</div>';document.querySelectorAll('.service-main').forEach(row=>row.addEventListener('click',()=>row.parentElement.classList.toggle('open')))};const renderEvents=p=>{const rows=p.data||[];document.querySelector('#event-count').textContent=`${p.total||0} 条记录`;document.querySelector('#events').innerHTML=rows.length?rows.map(e=>`<tr><td><span class="protocol">${esc(e.protocol||e.network||'tcp').toUpperCase()}</span></td><td>${esc(e.client_source||e.source_ip)}</td><td><b>${esc(e.destination)}</b></td><td class="${e.action==='block'?'blocked':''}">${esc(e.action)}${e.rule_tag?` / ${esc(e.rule_tag)}`:''}</td><td>${esc(new Date(e.occurred_at).toLocaleString())}</td></tr>`).join(''):'<tr><td colspan="5" class="empty">暂无匹配记录</td></tr>'};const refresh=async()=>{try{const [s,e]=await Promise.all([req('/summary',{range,group_by:group,metric}),req('/events',{range,destination:document.querySelector('#filter-destination').value,rule_tag:document.querySelector('#filter-rule').value,action:document.querySelector('#filter-action').value})]);renderSummary(s.data);renderEvents(e)}catch(e){if(e.message!=='auth')notice('加载失败：请确认插件已启用、采集器可上报，且管理员 Token 有效。',false)}};document.querySelector('#ranges').addEventListener('click',e=>{const r=e.target.dataset.range;if(!r)return;range=r;document.querySelectorAll('#ranges button').forEach(x=>x.classList.toggle('active',x===e.target));refresh()});document.querySelectorAll('[data-group]').forEach(b=>b.onclick=()=>{group=b.dataset.group;document.querySelectorAll('[data-group]').forEach(x=>x.classList.toggle('active',x===b));refresh()});document.querySelectorAll('[data-metric]').forEach(b=>b.onclick=()=>{metric=b.dataset.metric;document.querySelectorAll('[data-metric]').forEach(x=>x.classList.toggle('active',x===b));refresh()});document.querySelector('#refresh').onclick=refresh;document.querySelector('#apply').onclick=refresh;document.querySelector('#save-token').onclick=()=>{const t=document.querySelector('#token').value.trim();if(t){sessionStorage.setItem(saved,t);document.querySelector('#notice').style.display='none';document.querySelector('#token-row').style.display='none';refresh()}};document.querySelectorAll('[data-view]').forEach(b=>b.onclick=()=>{const view=b.dataset.view;document.querySelectorAll('[data-view]').forEach(x=>x.classList.toggle('active',x===b));document.querySelectorAll('.view').forEach(x=>x.classList.toggle('active',x.id===view));document.querySelector('#title').textContent=view==='analysis'?'流量分析':'流量审计记录';document.querySelector('#subtitle').textContent=view==='analysis'?'节点流量去向——按服务与目标地址分组。':'专线流量识别的实时记录，仅管理后台可见。'});refresh()})();
</script>
<script>
(() => {
  const auditTokenKey = 'xboard-risk-audit-admin-token';
  const acceptedName = /(?:^|[_-])(access[_-]?token|auth[_-]?token|admin[_-]?token|token)(?:$|[_-])/i;
  const plausibleToken = value => typeof value === 'string' && value.length >= 20 && !/^(?:null|undefined)$/i.test(value);
  const findToken = (value, key = '', depth = 0) => {
    if (depth > 5) return '';
    if (plausibleToken(value) && acceptedName.test(key)) return value.replace(/^Bearer\s+/i, '');
    if (typeof value !== 'object' || value === null) return '';
    for (const [childKey, childValue] of Object.entries(value)) {
      const found = findToken(childValue, childKey, depth + 1);
      if (found) return found;
    }
    return '';
  };
  const scanStore = store => {
    for (let index = 0; index < store.length; index++) {
      const key = store.key(index) || '';
      const raw = store.getItem(key) || '';
      if (plausibleToken(raw) && acceptedName.test(key)) return raw.replace(/^Bearer\s+/i, '');
      try {
        const found = findToken(JSON.parse(raw), key);
        if (found) return found;
      } catch (_) { /* non-JSON storage is ignored */ }
    }
    return '';
  };
  const detected = scanStore(sessionStorage) || scanStore(localStorage);
  if (detected) {
    sessionStorage.setItem(auditTokenKey, detected);
    document.querySelector('#notice').style.display = 'none';
    document.querySelector('#token-row').style.display = 'none';
    document.querySelector('#refresh').click();
  }
})();
</script>
<script>
(() => {
  const filters = document.querySelector('#audit .filters');
  const userFilter = document.createElement('input');
  userFilter.id = 'filter-user';
  userFilter.inputMode = 'numeric';
  userFilter.placeholder = '用户 ID';
  filters.prepend(userFilter);

  const header = document.querySelector('#audit .audit-table thead tr');
  const userHeader = document.createElement('th');
  userHeader.textContent = '用户';
  header.children[0].after(userHeader);

  let returnedUsers = [];
  const previousFetch = window.fetch.bind(window);
  window.fetch = async (input, init) => {
    const requestURL = new URL(input, location.origin);
    if (requestURL.pathname.endsWith('/risk-audit/events') || requestURL.pathname.endsWith('/risk-audit/summary')) {
      const userID = userFilter.value.trim();
      if (userID) requestURL.searchParams.set('user_id', userID);
    }
    const response = await previousFetch(requestURL, init);
    if (requestURL.pathname.endsWith('/risk-audit/events') && response.ok) {
      response.clone().json().then(payload => {
        returnedUsers = (payload.data || []).map(item => item.user_id || '—');
        queueMicrotask(renderUserColumn);
      }).catch(() => {});
    }
    return response;
  };

  function renderUserColumn() {
    const rows = document.querySelectorAll('#events tr');
    rows.forEach((row, index) => {
      if (row.dataset.auditUserRendered === 'true') return;
      const cells = row.querySelectorAll('td');
      if (cells.length !== 5 || !returnedUsers[index]) return;
      const cell = document.createElement('td');
      cell.textContent = returnedUsers[index] === '—' ? '—' : '#' + returnedUsers[index];
      cells[0].after(cell);
      row.dataset.auditUserRendered = 'true';
    });
  }

  new MutationObserver(() => queueMicrotask(renderUserColumn)).observe(document.querySelector('#events'), {childList: true});

  const analysis = document.querySelector('#analysis');
  const scope = document.createElement('div');
  scope.className = 'filters';
  scope.style.cssText = 'justify-content:flex-end;padding:0 0 18px;border:0';
  scope.innerHTML = '<label style="display:flex;align-items:center;gap:9px;color:#61748e">按用户查看 <input inputmode="numeric" placeholder="全部用户" aria-label="用户 ID"></label><button class="refresh" type="button">应用</button>';
  const scopeInput = scope.querySelector('input');
  scopeInput.value = userFilter.value;
  scopeInput.addEventListener('input', () => { userFilter.value = scopeInput.value; });
  scope.querySelector('button').addEventListener('click', () => document.querySelector('#apply').click());
  analysis.prepend(scope);
  document.querySelector('#refresh').click();
})();
</script>
</body>
</html>
