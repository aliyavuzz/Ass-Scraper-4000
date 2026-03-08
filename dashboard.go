package dashboard

// DashboardHTML is the complete self-contained HTML/CSS/JS dashboard
// for the Scrape-o-Matic 3000 web UI.
const DashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Scrape-o-Matic 3000</title>
<link href="https://unpkg.com/nes.css@2.3.0/css/nes.min.css" rel="stylesheet">
<link href="https://fonts.googleapis.com/css2?family=Press+Start+2P&display=swap" rel="stylesheet">
<style>
*,*::before,*::after{box-sizing:border-box}
html,body{margin:0;padding:0;min-height:100vh;background:#0a0a0a;color:#33ff33;font-family:"Press Start 2P",cursive;font-size:10px;overflow-x:hidden}
body::before{content:"";position:fixed;top:0;left:0;width:100%;height:100%;background:repeating-linear-gradient(0deg,rgba(0,0,0,0.15) 0px,rgba(0,0,0,0.15) 1px,transparent 1px,transparent 3px);pointer-events:none;z-index:9999}
body::after{content:"";position:fixed;top:0;left:0;width:100%;height:100%;box-shadow:inset 0 0 120px rgba(0,255,0,0.08),inset 0 0 60px rgba(0,255,0,0.04);pointer-events:none;z-index:9998}
.wrapper{max-width:1400px;margin:0 auto;padding:16px}
.header{text-align:center;padding:20px 0;border-bottom:3px solid #33ff33;margin-bottom:16px}
.header h1{color:#33ff33;text-shadow:0 0 10px #33ff33,0 0 20px #33ff33,0 0 40px #00aa00;font-size:18px;letter-spacing:2px}
.header .subtitle{color:#1a8a1a;font-size:8px;margin-top:8px}
.tab-bar{display:flex;gap:4px;margin-bottom:16px}
.tab-btn{background:#111;color:#1a8a1a;border:2px solid #1a5a1a;padding:10px 20px;font-family:"Press Start 2P",cursive;font-size:9px;cursor:pointer;transition:none}
.tab-btn:hover{background:#1a2a1a;color:#33ff33}
.tab-btn.active{background:#0d2b0d;color:#33ff33;border-color:#33ff33;text-shadow:0 0 8px #33ff33}
.tab-content{display:none}
.tab-content.active{display:block}
.nes-container{background:#0d0d0d !important;border-color:#1a5a1a !important;margin-bottom:16px}
.nes-container.is-dark{background:#0d0d0d !important}
.nes-container.with-title>.title{background-color:#0a0a0a !important;color:#33ff33 !important;font-size:9px}
.nes-table{background:#0a0a0a;width:100%;border-collapse:collapse}
.nes-table th{background:#111 !important;color:#33ff33 !important;font-size:8px;padding:8px 6px;border:1px solid #1a5a1a;text-align:left;white-space:nowrap}
.nes-table td{color:#22cc22;font-size:8px;padding:6px;border:1px solid #1a3a1a;vertical-align:middle;word-break:break-word}
.nes-table tr:hover{background:#0d1a0d}
.nes-btn{font-family:"Press Start 2P",cursive;font-size:8px}
.nes-btn.is-success{background:#1a5a1a;color:#33ff33;border-color:#33ff33}
.nes-btn.is-success:hover{background:#2a7a2a}
.nes-btn.is-error{background:#5a1a1a;color:#ff3333;border-color:#ff3333}
.nes-btn.is-warning{background:#5a5a1a;color:#ffff33;border-color:#ffff33}
.nes-input,.nes-select select,.nes-checkbox{font-family:"Press Start 2P",cursive;font-size:8px}
input[type="text"],input[type="number"],select{background:#080808 !important;color:#33ff33 !important;border:2px solid #1a5a1a !important;padding:6px 8px;font-family:"Press Start 2P",cursive;font-size:8px;width:100%}
input[type="text"]:focus,select:focus{border-color:#33ff33 !important;outline:none;box-shadow:0 0 8px rgba(51,255,51,0.3)}
select option{background:#0a0a0a;color:#33ff33}
label{color:#1a8a1a;font-size:8px;display:block;margin-bottom:4px}
.badge{display:inline-block;padding:2px 8px;font-size:7px;font-family:"Press Start 2P",cursive;border:1px solid;text-transform:uppercase}
.badge-idle{color:#666;border-color:#444;background:#111}
.badge-running{color:#33ff33;border-color:#33ff33;background:#0d2b0d;animation:pulse-badge 1.2s ease-in-out infinite}
.badge-scheduled{color:#ffff33;border-color:#ffff33;background:#2b2b0d;animation:blink-badge 1s step-end infinite}
.badge-done{color:#33ccff;border-color:#33ccff;background:#0d1a2b}
.badge-error{color:#ff3333;border-color:#ff3333;background:#2b0d0d}
@keyframes pulse-badge{0%,100%{opacity:1}50%{opacity:0.5}}
@keyframes blink-badge{0%,100%{opacity:1}50%{opacity:0}}
.loading-bar{font-family:monospace;font-size:14px;color:#33ff33;letter-spacing:2px;text-shadow:0 0 8px #33ff33;height:20px;overflow:hidden}
.terminal{background:#020802;border:2px solid #1a3a1a;padding:12px;font-family:"Courier New",monospace;font-size:10px;color:#33ff33;height:500px;overflow-y:auto;line-height:1.6;text-shadow:0 0 4px #33ff33,0 0 8px rgba(0,255,0,0.3);white-space:pre-wrap;word-break:break-all}
.terminal .log-line{margin:0;padding:1px 0;transition:color 2s ease-out}
.terminal .log-line.fresh{color:#66ff66;text-shadow:0 0 8px #66ff66,0 0 16px rgba(0,255,0,0.5)}
.terminal .sess-id{color:#33ccff;font-weight:bold}
.terminal .timestamp{color:#888}
.terminal .evt-status{color:#ffff33}
.terminal .evt-error{color:#ff3333}
.grid-2{display:grid;grid-template-columns:1fr 1fr;gap:16px}
.grid-3{display:grid;grid-template-columns:1fr 1fr 1fr;gap:12px}
.gap-8{gap:8px}
.flex{display:flex}
.flex-wrap{flex-wrap:wrap}
.items-center{align-items:center}
.gap-12{gap:12px}
.mt-8{margin-top:8px}
.mt-12{margin-top:12px}
.mt-16{margin-top:16px}
.mb-8{margin-bottom:8px}
.mb-12{margin-bottom:12px}
.p-8{padding:8px}
.p-12{padding:12px}
.text-center{text-align:center}
.hidden{display:none !important}
.full-width{width:100%}
.screenshot-area{background:#050505;border:2px solid #1a3a1a;min-height:100px;display:flex;align-items:center;justify-content:center;padding:8px}
.screenshot-area img{max-width:100%;border:1px solid #1a5a1a}
.screenshot-area .placeholder{color:#333;font-size:8px}
.curl-box{background:#111;border:1px solid #1a3a1a;padding:8px;font-family:"Courier New",monospace;font-size:8px;color:#1a8a1a;word-break:break-all;position:relative;cursor:pointer}
.curl-box:hover{border-color:#33ff33;background:#0d1a0d}
.curl-box .copy-hint{position:absolute;top:2px;right:6px;font-size:6px;color:#444}
.curl-box:active .copy-hint{color:#33ff33}
.copied-toast{position:fixed;bottom:20px;right:20px;background:#0d2b0d;border:2px solid #33ff33;color:#33ff33;padding:10px 16px;font-family:"Press Start 2P",cursive;font-size:8px;z-index:10000;display:none;text-shadow:0 0 6px #33ff33}
.twofa-section{background:#1a1a00;border:2px solid #aaaa00;padding:12px;margin-top:12px}
.twofa-section label{color:#ffff33}
.data-table-wrap{overflow-x:auto}
.data-table-wrap table{min-width:800px}
.filter-bar{display:flex;gap:8px;align-items:center;margin-bottom:12px}
.filter-bar label{margin:0}
.filter-bar select{width:auto;min-width:120px}
.cron-row{display:flex;gap:8px;align-items:flex-end;flex-wrap:wrap}
.cron-row input{flex:1;min-width:140px}
.toggle-switch{display:inline-flex;align-items:center;gap:6px;cursor:pointer;font-size:8px;color:#1a8a1a}
.toggle-switch input{display:none}
.toggle-track{width:36px;height:18px;background:#333;border:2px solid #555;position:relative;display:inline-block}
.toggle-track::after{content:"";position:absolute;width:10px;height:10px;background:#555;top:2px;left:2px;transition:none}
.toggle-switch input:checked+.toggle-track{background:#0d2b0d;border-color:#33ff33}
.toggle-switch input:checked+.toggle-track::after{background:#33ff33;left:20px}
.var-inputs{display:grid;grid-template-columns:1fr 1fr;gap:8px}
.var-group label{font-size:7px;color:#1a8a1a;margin-bottom:2px}
.api-section{margin-top:16px}
.api-row{margin-bottom:12px}
.api-row .api-label{font-size:8px;color:#33ff33;margin-bottom:4px;display:flex;align-items:center;gap:8px}
.api-row .api-method{font-size:7px;padding:1px 6px;border:1px solid;display:inline-block}
.api-row .api-method.get{color:#33ccff;border-color:#33ccff}
.api-row .api-method.post{color:#33ff33;border-color:#33ff33}
.api-row .api-method.put{color:#ffff33;border-color:#ffff33}
.checkbox-row{display:flex;gap:16px;align-items:center;margin:8px 0}
.checkbox-row label{display:inline-flex;align-items:center;gap:6px;cursor:pointer;margin:0}
.checkbox-row input[type="checkbox"]{width:14px;height:14px;accent-color:#33ff33}
@media(max-width:900px){.grid-2{grid-template-columns:1fr}.grid-3{grid-template-columns:1fr}.var-inputs{grid-template-columns:1fr}}
</style>
</head>
<body>
<div class="wrapper">
<div class="header">
<h1>SCRAPE-O-MATIC 3000</h1>
<div class="subtitle">[ RETRO WEB AUTOMATION DASHBOARD ]</div>
</div>

<div class="tab-bar">
<button class="tab-btn active" data-tab="gallery">GALLERY</button>
<button class="tab-btn" data-tab="matrix">LIVE MATRIX</button>
<button class="tab-btn" data-tab="datalab">DATA LAB</button>
</div>

<!-- ================================================================== -->
<!-- TAB 1: RECIPE GALLERY                                              -->
<!-- ================================================================== -->
<div id="tab-gallery" class="tab-content active">

<div class="nes-container is-dark with-title">
<p class="title">RECIPE INVENTORY</p>
<div style="overflow-x:auto">
<table class="nes-table" id="recipe-table">
<thead><tr>
<th>NAME</th><th>SITE</th><th>STATUS</th><th>SCHEDULE</th><th>STEPS</th>
</tr></thead>
<tbody id="recipe-tbody"><tr><td colspan="5" class="text-center" style="color:#444">Loading recipes...</td></tr></tbody>
</table>
</div>
</div>

<div class="grid-2">
<div>
<div class="nes-container is-dark with-title">
<p class="title">LAUNCH PANEL</p>
<div class="mb-8">
<label>RECIPE</label>
<select id="recipe-select"><option value="">-- select recipe --</option></select>
</div>
<div id="var-container" class="var-inputs mb-8"></div>
<div class="checkbox-row">
<label><input type="checkbox" id="chk-headless" checked> Headless</label>
<label><input type="checkbox" id="chk-screenshot"> Screenshot</label>
</div>
<div class="flex gap-8 mt-8">
<button class="nes-btn is-success" id="btn-run" disabled>RUN</button>
<button class="nes-btn is-error" id="btn-stop" disabled>STOP</button>
</div>
<div id="run-loading" class="loading-bar mt-8 hidden"></div>
<div id="run-status" class="mt-8" style="font-size:8px;color:#1a8a1a"></div>

<div id="twofa-section" class="twofa-section hidden mt-12">
<label>2FA CODE REQUIRED</label>
<div class="flex gap-8 mt-8">
<input type="text" id="twofa-input" placeholder="Enter 2FA code" style="flex:1">
<button class="nes-btn is-warning" id="btn-2fa">SEND</button>
</div>
</div>
</div>

<div class="nes-container is-dark with-title mt-12">
<p class="title">CRON SCHEDULING</p>
<div class="cron-row">
<div style="flex:1">
<label>CRON EXPRESSION</label>
<input type="text" id="cron-input" placeholder="*/30 * * * *">
</div>
<button class="nes-btn is-success" id="btn-set-cron">SET SCHEDULE</button>
</div>
<div class="mt-8">
<label class="toggle-switch">
<input type="checkbox" id="cron-enabled" checked>
<span class="toggle-track"></span>
ENABLED
</label>
</div>
</div>
</div>

<div>
<div class="nes-container is-dark with-title">
<p class="title">SCREENSHOT</p>
<div class="screenshot-area" id="screenshot-area">
<span class="placeholder">No screenshot captured</span>
</div>
</div>

<div class="nes-container is-dark with-title">
<p class="title">EXECUTION LOG</p>
<div class="terminal" id="gallery-terminal" style="height:200px"></div>
</div>
</div>
</div>

<div class="nes-container is-dark with-title api-section">
<p class="title">API DASHBOARD</p>
<div class="api-row">
<div class="api-label"><span class="api-method get">GET</span> /api/v1/recipes</div>
<div class="curl-box" onclick="copyCurl(this)">curl -H "X-API-Key: YOUR_KEY" http://localhost:PORT/api/v1/recipes<span class="copy-hint">[click to copy]</span></div>
</div>
<div class="api-row">
<div class="api-label"><span class="api-method post">POST</span> /api/v1/recipes/{name}/run</div>
<div class="curl-box" onclick="copyCurl(this)">curl -X POST -H "Content-Type: application/json" -H "Accept: text/event-stream" -H "X-API-Key: YOUR_KEY" -d '{"variables":{},"screenshot":true}' http://localhost:PORT/api/v1/recipes/RECIPE_NAME/run<span class="copy-hint">[click to copy]</span></div>
</div>
<div class="api-row">
<div class="api-label"><span class="api-method get">GET</span> /api/v1/sessions/{id}/events (SSE)</div>
<div class="curl-box" onclick="copyCurl(this)">curl -H "Accept: text/event-stream" -H "X-API-Key: YOUR_KEY" http://localhost:PORT/api/v1/sessions/SESSION_ID/events<span class="copy-hint">[click to copy]</span></div>
</div>
<div class="api-row">
<div class="api-label"><span class="api-method post">POST</span> /api/v1/sessions/{id}/2fa</div>
<div class="curl-box" onclick="copyCurl(this)">curl -X POST -H "Content-Type: application/json" -H "X-API-Key: YOUR_KEY" -d '{"code":"123456"}' http://localhost:PORT/api/v1/sessions/SESSION_ID/2fa<span class="copy-hint">[click to copy]</span></div>
</div>
<div class="api-row">
<div class="api-label"><span class="api-method get">GET</span> /api/v1/recipes/{name}/data?history=N</div>
<div class="curl-box" onclick="copyCurl(this)">curl -H "X-API-Key: YOUR_KEY" http://localhost:PORT/api/v1/recipes/RECIPE_NAME/data?history=20<span class="copy-hint">[click to copy]</span></div>
</div>
<div class="api-row">
<div class="api-label"><span class="api-method put">PUT</span> /api/v1/recipes/{name}/schedule</div>
<div class="curl-box" onclick="copyCurl(this)">curl -X PUT -H "Content-Type: application/json" -H "X-API-Key: YOUR_KEY" -d '{"cron":"*/30 * * * *","enabled":true}' http://localhost:PORT/api/v1/recipes/RECIPE_NAME/schedule<span class="copy-hint">[click to copy]</span></div>
</div>
</div>
</div>

<!-- ================================================================== -->
<!-- TAB 2: LIVE MATRIX                                                 -->
<!-- ================================================================== -->
<div id="tab-matrix" class="tab-content">
<div class="nes-container is-dark with-title">
<p class="title">LIVE EVENT MATRIX</p>
<div class="terminal" id="matrix-terminal" style="height:calc(100vh - 220px);min-height:400px"></div>
</div>
</div>

<!-- ================================================================== -->
<!-- TAB 3: DATA LAB                                                    -->
<!-- ================================================================== -->
<div id="tab-datalab" class="tab-content">
<div class="nes-container is-dark with-title">
<p class="title">DATA LAB</p>
<div class="filter-bar">
<label>RECIPE:</label>
<select id="datalab-recipe" style="width:200px"><option value="">-- select --</option></select>
<label>ROWS:</label>
<select id="datalab-limit" style="width:100px">
<option value="10">Last 10</option>
<option value="20" selected>Last 20</option>
<option value="50">Last 50</option>
<option value="100">Last 100</option>
</select>
<button class="nes-btn is-success" id="btn-datalab-load">LOAD</button>
</div>
<div class="data-table-wrap">
<table class="nes-table" id="datalab-table">
<thead><tr><th>TIMESTAMP</th><th>STATUS</th><th>DURATION</th><th>DATA</th></tr></thead>
<tbody id="datalab-tbody"><tr><td colspan="4" class="text-center" style="color:#444">Select a recipe and click LOAD</td></tr></tbody>
</table>
</div>
</div>
</div>

</div>
<div class="copied-toast" id="copied-toast">COPIED TO CLIPBOARD!</div>

<script>
(function(){
"use strict";

// ====================================================================
// STATE
// ====================================================================
var recipes = [];
var currentSession = null;
var currentEventSource = null;
var matrixSources = {};
var loadingInterval = null;

// ====================================================================
// TABS
// ====================================================================
var tabBtns = document.querySelectorAll(".tab-btn");
tabBtns.forEach(function(btn){
  btn.addEventListener("click", function(){
    tabBtns.forEach(function(b){ b.classList.remove("active"); });
    btn.classList.add("active");
    document.querySelectorAll(".tab-content").forEach(function(tc){ tc.classList.remove("active"); });
    document.getElementById("tab-" + btn.getAttribute("data-tab")).classList.add("active");
  });
});

// ====================================================================
// HELPERS
// ====================================================================
function escapeHtml(str){
  if(!str) return "";
  return String(str).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;");
}

function apiGet(url, cb){
  var xhr = new XMLHttpRequest();
  xhr.open("GET", url);
  xhr.setRequestHeader("Accept","application/json");
  xhr.onload = function(){
    if(xhr.status >= 200 && xhr.status < 300){
      try{ cb(null, JSON.parse(xhr.responseText)); }
      catch(e){ cb(e, null); }
    } else { cb(new Error("HTTP " + xhr.status), null); }
  };
  xhr.onerror = function(){ cb(new Error("Network error"), null); };
  xhr.send();
}

function apiPost(url, data, cb){
  var xhr = new XMLHttpRequest();
  xhr.open("POST", url);
  xhr.setRequestHeader("Content-Type","application/json");
  xhr.setRequestHeader("Accept","text/event-stream");
  xhr.onload = function(){
    try{ cb(null, JSON.parse(xhr.responseText)); }
    catch(e){ cb(e, null); }
  };
  xhr.onerror = function(){ cb(new Error("Network error"), null); };
  xhr.send(JSON.stringify(data));
}

function apiPut(url, data, cb){
  var xhr = new XMLHttpRequest();
  xhr.open("PUT", url);
  xhr.setRequestHeader("Content-Type","application/json");
  xhr.onload = function(){
    try{ cb(null, JSON.parse(xhr.responseText)); }
    catch(e){ cb(e, null); }
  };
  xhr.onerror = function(){ cb(new Error("Network error"), null); };
  xhr.send(JSON.stringify(data));
}

function nowStamp(){
  var d = new Date();
  var hh = String(d.getHours()).padStart(2,"0");
  var mm = String(d.getMinutes()).padStart(2,"0");
  var ss = String(d.getSeconds()).padStart(2,"0");
  return hh+":"+mm+":"+ss;
}

// ====================================================================
// RECIPE TABLE
// ====================================================================
function loadRecipes(){
  apiGet("/api/v1/recipes", function(err, data){
    if(err || !Array.isArray(data)){
      document.getElementById("recipe-tbody").innerHTML = '<tr><td colspan="5" style="color:#ff3333">Failed to load recipes</td></tr>';
      return;
    }
    recipes = data;
    renderRecipeTable();
    populateSelects();
  });
}

function renderRecipeTable(){
  var tbody = document.getElementById("recipe-tbody");
  if(!recipes.length){
    tbody.innerHTML = '<tr><td colspan="5" style="color:#444">No recipes found. Create one with: scrape learn</td></tr>';
    return;
  }
  var html = "";
  recipes.forEach(function(r){
    var statusClass = "badge-idle";
    var statusText = "IDLE";
    if(currentSession && currentSession.recipe === r.name){
      if(currentSession.status === "running" || currentSession.status === "waiting_2fa"){
        statusClass = "badge-running";
        statusText = "RUNNING...";
      } else if(currentSession.status === "done"){
        statusClass = "badge-done";
        statusText = "DONE";
      } else if(currentSession.status === "error"){
        statusClass = "badge-error";
        statusText = "ERROR";
      }
    }
    if(r.cron_expr && r.enabled && statusClass === "badge-idle"){
      statusClass = "badge-scheduled";
      statusText = "SCHEDULED";
    }
    var schedule = r.cron_expr || "\u2014";
    html += "<tr>";
    html += "<td>" + escapeHtml(r.name) + "</td>";
    html += "<td>" + escapeHtml(r.site) + "</td>";
    html += '<td><span class="badge ' + statusClass + '">' + statusText + "</span></td>";
    html += "<td>" + escapeHtml(schedule) + "</td>";
    html += "<td>" + (r.steps || 0) + "</td>";
    html += "</tr>";
  });
  tbody.innerHTML = html;
}

function populateSelects(){
  var sel1 = document.getElementById("recipe-select");
  var sel2 = document.getElementById("datalab-recipe");
  var cur1 = sel1.value;
  var cur2 = sel2.value;
  sel1.innerHTML = '<option value="">-- select recipe --</option>';
  sel2.innerHTML = '<option value="">-- select --</option>';
  recipes.forEach(function(r){
    sel1.innerHTML += '<option value="' + escapeHtml(r.name) + '">' + escapeHtml(r.name) + "</option>";
    sel2.innerHTML += '<option value="' + escapeHtml(r.name) + '">' + escapeHtml(r.name) + "</option>";
  });
  if(cur1) sel1.value = cur1;
  if(cur2) sel2.value = cur2;
}

// ====================================================================
// VARIABLE INPUTS
// ====================================================================
document.getElementById("recipe-select").addEventListener("change", function(){
  var name = this.value;
  var container = document.getElementById("var-container");
  container.innerHTML = "";
  document.getElementById("btn-run").disabled = !name;
  if(!name) return;
  var recipe = recipes.find(function(r){ return r.name === name; });
  if(!recipe || !recipe.variables || !recipe.variables.length) return;
  recipe.variables.forEach(function(v){
    var div = document.createElement("div");
    div.className = "var-group";
    div.innerHTML = '<label>' + escapeHtml(v.toUpperCase()) + '</label><input type="text" class="var-input" data-var="' + escapeHtml(v) + '" placeholder="Enter ' + escapeHtml(v) + '">';
    container.appendChild(div);
  });
  // Load existing schedule
  apiGet("/api/v1/recipes/" + encodeURIComponent(name) + "/schedule", function(err, data){
    if(!err && data){
      document.getElementById("cron-input").value = data.cron || "";
      document.getElementById("cron-enabled").checked = data.enabled !== false;
    }
  });
});

// ====================================================================
// RUN / STOP
// ====================================================================
var loadingChars = ["\u2591","\u2592","\u2593","\u2588"];
function startLoadingAnim(){
  var el = document.getElementById("run-loading");
  el.classList.remove("hidden");
  var frame = 0;
  loadingInterval = setInterval(function(){
    var bar = "";
    for(var i=0;i<20;i++){
      bar += loadingChars[(frame + i) % loadingChars.length];
    }
    el.textContent = bar;
    frame++;
  }, 150);
}
function stopLoadingAnim(){
  if(loadingInterval){ clearInterval(loadingInterval); loadingInterval = null; }
  document.getElementById("run-loading").classList.add("hidden");
}

document.getElementById("btn-run").addEventListener("click", function(){
  var recipeName = document.getElementById("recipe-select").value;
  if(!recipeName) return;

  var variables = {};
  document.querySelectorAll(".var-input").forEach(function(inp){
    variables[inp.getAttribute("data-var")] = inp.value;
  });
  var screenshot = document.getElementById("chk-screenshot").checked;

  var postData = { variables: variables, screenshot: screenshot, headless: document.getElementById("chk-headless").checked };

  document.getElementById("btn-run").disabled = true;
  document.getElementById("btn-stop").disabled = false;
  document.getElementById("run-status").textContent = "Starting recipe...";
  document.getElementById("gallery-terminal").innerHTML = "";
  document.getElementById("twofa-section").classList.add("hidden");
  startLoadingAnim();

  apiPost("/api/v1/recipes/" + encodeURIComponent(recipeName) + "/run", postData, function(err, data){
    if(err || !data || !data.session_id){
      document.getElementById("run-status").textContent = "Failed to start: " + (err ? err.message : "unknown error");
      stopLoadingAnim();
      document.getElementById("btn-run").disabled = false;
      document.getElementById("btn-stop").disabled = true;
      return;
    }
    currentSession = { id: data.session_id, recipe: recipeName, status: "running" };
    renderRecipeTable();
    connectSSE(data.session_id, recipeName);
  });
});

document.getElementById("btn-stop").addEventListener("click", function(){
  if(currentEventSource){ currentEventSource.close(); currentEventSource = null; }
  if(currentSession){ currentSession.status = "stopped"; }
  stopLoadingAnim();
  document.getElementById("btn-run").disabled = false;
  document.getElementById("btn-stop").disabled = true;
  document.getElementById("run-status").textContent = "Stopped by user.";
  renderRecipeTable();
});

function connectSSE(sessionId, recipeName){
  if(currentEventSource) currentEventSource.close();
  var es = new EventSource("/api/v1/sessions/" + sessionId + "/events");
  currentEventSource = es;

  // Also track in matrix
  addMatrixSource(sessionId, es);

  es.addEventListener("log", function(e){
    appendToTerminal("gallery-terminal", e.data);
    appendToMatrix(sessionId, "log", e.data);
  });

  es.addEventListener("status", function(e){
    var st = e.data;
    document.getElementById("run-status").textContent = "Status: " + st;
    if(currentSession) currentSession.status = st;
    renderRecipeTable();
    appendToMatrix(sessionId, "status", st);

    if(st === "waiting_2fa"){
      document.getElementById("twofa-section").classList.remove("hidden");
    } else {
      document.getElementById("twofa-section").classList.add("hidden");
    }

    if(st === "done" || st === "error"){
      stopLoadingAnim();
      document.getElementById("btn-run").disabled = false;
      document.getElementById("btn-stop").disabled = true;
    }
  });

  es.addEventListener("result", function(e){
    stopLoadingAnim();
    document.getElementById("btn-run").disabled = false;
    document.getElementById("btn-stop").disabled = true;
    try{
      var result = JSON.parse(e.data);
      if(result.screenshot){
        var area = document.getElementById("screenshot-area");
        area.innerHTML = '<img src="' + result.screenshot + '" alt="Screenshot">';
      }
      if(result.success){
        document.getElementById("run-status").textContent = "Completed successfully in " + (result.duration || "?");
        appendToTerminal("gallery-terminal", "=== RECIPE COMPLETE === Duration: " + (result.duration || "?"));
      } else {
        document.getElementById("run-status").textContent = "Error: " + (result.error || "unknown");
        appendToTerminal("gallery-terminal", "=== ERROR: " + (result.error || "unknown") + " ===");
      }
    } catch(ex){}
    es.close();
    currentEventSource = null;
    appendToMatrix(sessionId, "result", "Session ended");
  });

  es.onerror = function(){
    stopLoadingAnim();
    document.getElementById("btn-run").disabled = false;
    document.getElementById("btn-stop").disabled = true;
    es.close();
    currentEventSource = null;
  };
}

// ====================================================================
// 2FA
// ====================================================================
document.getElementById("btn-2fa").addEventListener("click", function(){
  if(!currentSession) return;
  var code = document.getElementById("twofa-input").value.trim();
  if(!code) return;
  var xhr = new XMLHttpRequest();
  xhr.open("POST", "/api/v1/sessions/" + currentSession.id + "/2fa");
  xhr.setRequestHeader("Content-Type","application/json");
  xhr.send(JSON.stringify({code: code}));
  document.getElementById("twofa-input").value = "";
});

// ====================================================================
// CRON SCHEDULING
// ====================================================================
document.getElementById("btn-set-cron").addEventListener("click", function(){
  var recipeName = document.getElementById("recipe-select").value;
  if(!recipeName) return;
  var cron = document.getElementById("cron-input").value.trim();
  var enabled = document.getElementById("cron-enabled").checked;
  apiPut("/api/v1/recipes/" + encodeURIComponent(recipeName) + "/schedule", {cron: cron, enabled: enabled}, function(err, data){
    if(err){
      document.getElementById("run-status").textContent = "Schedule error: " + err.message;
    } else {
      document.getElementById("run-status").textContent = "Schedule updated: " + (cron || "cleared");
      loadRecipes();
    }
  });
});

// ====================================================================
// TERMINAL HELPERS
// ====================================================================
function appendToTerminal(termId, text){
  var term = document.getElementById(termId);
  var line = document.createElement("div");
  line.className = "log-line fresh";
  line.textContent = text;
  term.appendChild(line);
  term.scrollTop = term.scrollHeight;
  setTimeout(function(){ line.classList.remove("fresh"); }, 2000);
}

// ====================================================================
// LIVE MATRIX
// ====================================================================
function addMatrixSource(sessionId, eventSource){
  matrixSources[sessionId] = eventSource;
}

function appendToMatrix(sessionId, eventType, text){
  var term = document.getElementById("matrix-terminal");
  var line = document.createElement("div");
  line.className = "log-line fresh";
  var prefix = '<span class="sess-id">[' + escapeHtml(sessionId) + ']</span> <span class="timestamp">' + nowStamp() + '</span> ';
  if(eventType === "status"){
    prefix += '<span class="evt-status">[STATUS]</span> ';
  } else if(eventType === "error"){
    prefix += '<span class="evt-error">[ERROR]</span> ';
  } else {
    prefix += "[LOG] ";
  }
  line.innerHTML = prefix + escapeHtml(text);
  term.appendChild(line);
  term.scrollTop = term.scrollHeight;
  setTimeout(function(){ line.classList.remove("fresh"); }, 2000);

  // Trim old lines from matrix if over 500
  while(term.children.length > 500){
    term.removeChild(term.firstChild);
  }
}

// ====================================================================
// DATA LAB
// ====================================================================
document.getElementById("btn-datalab-load").addEventListener("click", loadDataLab);

function loadDataLab(){
  var recipe = document.getElementById("datalab-recipe").value;
  var limit = document.getElementById("datalab-limit").value;
  var tbody = document.getElementById("datalab-tbody");
  if(!recipe){
    tbody.innerHTML = '<tr><td colspan="4" style="color:#ff3333">Select a recipe first</td></tr>';
    return;
  }
  tbody.innerHTML = '<tr><td colspan="4" style="color:#444">Loading data...</td></tr>';
  apiGet("/api/v1/recipes/" + encodeURIComponent(recipe) + "/data?history=" + limit, function(err, data){
    if(err || !Array.isArray(data)){
      tbody.innerHTML = '<tr><td colspan="4" style="color:#ff3333">No data available</td></tr>';
      return;
    }
    if(!data.length){
      tbody.innerHTML = '<tr><td colspan="4" style="color:#444">No scrape data for this recipe</td></tr>';
      return;
    }
    var html = "";
    data.forEach(function(row){
      var statusBadge = row.success ? '<span class="badge badge-done">OK</span>' : '<span class="badge badge-error">ERR</span>';
      var dataStr = "";
      try{
        var parsed = JSON.parse(row.data);
        var keys = Object.keys(parsed);
        var preview = [];
        keys.slice(0,4).forEach(function(k){
          var val = String(parsed[k]);
          if(val.length > 40) val = val.substring(0,37) + "...";
          preview.push(escapeHtml(k) + ": " + escapeHtml(val));
        });
        dataStr = preview.join(" | ");
        if(keys.length > 4) dataStr += " (+"+( keys.length - 4) + " more)";
      } catch(e){
        dataStr = escapeHtml(row.data).substring(0,100);
      }
      if(row.error && !row.success){
        dataStr = '<span style="color:#ff3333">' + escapeHtml(row.error) + "</span>";
      }
      html += "<tr>";
      html += "<td style='white-space:nowrap'>" + escapeHtml(row.created_at) + "</td>";
      html += "<td>" + statusBadge + "</td>";
      html += "<td>" + escapeHtml(row.duration || "\u2014") + "</td>";
      html += "<td>" + dataStr + "</td>";
      html += "</tr>";
    });
    tbody.innerHTML = html;
  });
}

// ====================================================================
// CURL COPY
// ====================================================================
window.copyCurl = function(el){
  var text = el.textContent.replace("[click to copy]","").trim();
  text = text.replace("PORT", window.location.port || "8080");
  if(navigator.clipboard){
    navigator.clipboard.writeText(text);
  } else {
    var ta = document.createElement("textarea");
    ta.value = text;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand("copy");
    document.body.removeChild(ta);
  }
  var toast = document.getElementById("copied-toast");
  toast.style.display = "block";
  setTimeout(function(){ toast.style.display = "none"; }, 1500);
};

// ====================================================================
// INIT
// ====================================================================
loadRecipes();
setInterval(loadRecipes, 15000);

})();
</script>
</body>
</html>`
