package phoneintake

import "fmt"

func phonePage(token string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
<title>Trace Intake</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
html,body{height:100%%;overflow:hidden;touch-action:manipulation}
body{
  font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
  background:#0f1117;color:#e2e5f0;
}
/* -------- Scanner -------- */
#scanner-view{position:fixed;inset:0;background:#000;display:flex;flex-direction:column}
#camera{width:100%%;height:100%%;object-fit:cover}
.scan-overlay{
  position:absolute;inset:0;display:flex;align-items:center;justify-content:center;
  pointer-events:none;
}
.scan-frame{
  width:260px;height:260px;
  border:2px solid rgba(59,130,246,.6);border-radius:16px;
  box-shadow:0 0 0 9999px rgba(0,0,0,.45);
}
.scan-label{
  position:absolute;bottom:0;left:0;right:0;
  text-align:center;padding:24px 16px;
  font-size:15px;color:#8b90a0;
  background:linear-gradient(transparent,rgba(15,17,23,.85));
}
.scan-label.error{color:#f87171}
.scanner-paused #camera{filter:brightness(.35)}

/* -------- Bottom sheet -------- */
.sheet{
  position:fixed;left:0;right:0;bottom:0;
  background:#1a1d27;border-top:1px solid #2a2e3e;
  border-radius:14px 14px 0 0;
  padding:20px 16px calc(env(safe-area-inset-bottom,0px) + 16px);
  transform:translateY(100%%);transition:transform .28s ease;
  z-index:10;max-height:92vh;overflow-y:auto;
}
.sheet.visible{transform:translateY(0)}

/* -------- Component info -------- */
.comp-header{display:flex;gap:14px;margin-bottom:16px}
.comp-image{
  width:72px;height:72px;border-radius:8px;
  background:#141720;object-fit:contain;flex-shrink:0;
  border:1px solid #2a2e3e;
}
.comp-image.hidden{display:none}
.comp-title{font-size:15px;font-weight:600;color:#e2e5f0;margin-bottom:4px;word-break:break-word}
.comp-sub{font-size:12px;color:#8b90a0;line-height:1.5}
.info-grid{
  display:grid;grid-template-columns:auto 1fr;gap:4px 12px;
  font-size:13px;margin-bottom:16px;
}
.info-label{color:#5c6070;white-space:nowrap}
.info-value{color:#e2e5f0;word-break:break-all}

/* -------- Quantity controls -------- */
.mode-toggle{display:flex;gap:0;margin-bottom:12px;border-radius:6px;overflow:hidden;border:1px solid #2a2e3e}
.mode-btn{
  flex:1;padding:8px 0;font-size:13px;font-weight:500;
  background:#141720;color:#8b90a0;border:none;
  transition:background .12s,color .12s;cursor:pointer;
}
.mode-btn.active{background:rgba(59,130,246,.15);color:#7cb3ff}
.quick-btns{display:flex;gap:8px;margin-bottom:10px}
.quick-btn{
  flex:1;padding:12px 0;font-size:16px;font-weight:600;
  background:#141720;color:#e2e5f0;border:1px solid #2a2e3e;
  border-radius:8px;cursor:pointer;
  transition:background .1s;
}
.quick-btn:active{background:#222637}
.qty-input-row{display:flex;gap:8px;margin-bottom:14px;align-items:center}
.qty-input{
  flex:1;padding:12px;font-size:18px;
  background:#141720;color:#e2e5f0;
  border:1px solid #2a2e3e;border-radius:8px;
  text-align:center;
  -moz-appearance:textfield;
}
.qty-input::-webkit-inner-spin-button,.qty-input::-webkit-outer-spin-button{-webkit-appearance:none}
.qty-input:focus{outline:none;border-color:#3b82f6}
.qty-label{font-size:12px;color:#5c6070;text-align:center;margin-bottom:4px}

.submit-btn{
  width:100%%;padding:14px;font-size:15px;font-weight:600;
  background:#3b82f6;color:#fff;border:none;border-radius:8px;
  cursor:pointer;transition:background .12s;margin-bottom:8px;
}
.submit-btn:active{background:#2563eb}
.submit-btn:disabled{opacity:.5;cursor:default}
.cancel-btn{
  width:100%%;padding:10px;font-size:13px;font-weight:500;
  background:transparent;color:#8b90a0;border:none;
  cursor:pointer;
}
.cancel-btn:active{color:#e2e5f0}

/* -------- Unresolved -------- */
.unresolved-icon{font-size:32px;text-align:center;margin-bottom:8px;color:#fbbf24}
.unresolved-title{font-size:15px;font-weight:600;text-align:center;margin-bottom:12px;color:#fcd34d}
.raw-qr{
  font-family:'SF Mono','Cascadia Code','Fira Code',monospace;
  font-size:12px;background:#141720;border:1px solid #2a2e3e;
  border-radius:6px;padding:10px;word-break:break-all;
  color:#8b90a0;margin-bottom:16px;max-height:80px;overflow-y:auto;
}

/* -------- Success toast -------- */
.toast{
  position:fixed;top:0;left:0;right:0;
  background:rgba(52,211,153,.15);
  border-bottom:1px solid rgba(52,211,153,.3);
  color:#6ee7b7;font-size:14px;font-weight:600;
  text-align:center;padding:14px;
  transform:translateY(-100%%);transition:transform .25s ease;
  z-index:20;
}
.toast.visible{transform:translateY(0)}
</style>
</head>
<body>

<div id="scanner-view">
  <video id="camera" autoplay playsinline muted></video>
  <div class="scan-overlay"><div class="scan-frame"></div></div>
  <div class="scan-label" id="scan-label">Point camera at a QR code</div>
</div>

<div class="sheet" id="result-sheet">
  <div class="comp-header">
    <img class="comp-image hidden" id="comp-image" alt="Product" />
    <div>
      <div class="comp-title" id="comp-title"></div>
      <div class="comp-sub" id="comp-sub"></div>
    </div>
  </div>
  <div class="info-grid" id="info-grid"></div>

  <div class="mode-toggle">
    <button class="mode-btn active" id="mode-delta" onclick="setMode('delta')">Add</button>
    <button class="mode-btn" id="mode-set" onclick="setMode('set')">Set exact</button>
  </div>
  <div class="quick-btns" id="quick-btns">
    <button class="quick-btn" onclick="quickAdd(1)">+1</button>
    <button class="quick-btn" onclick="quickAdd(5)">+5</button>
    <button class="quick-btn" onclick="quickAdd(10)">+10</button>
  </div>
  <div class="qty-label" id="qty-label">Add to current count</div>
  <div class="qty-input-row">
    <input type="number" class="qty-input" id="qty-input" inputmode="numeric" pattern="[0-9]*" value="0" min="0" />
  </div>
  <button class="submit-btn" id="submit-btn" onclick="submitCount()">Submit</button>
  <button class="cancel-btn" onclick="cancelResult()">Cancel</button>
</div>

<div class="sheet" id="unresolved-sheet">
  <div class="unresolved-icon">?</div>
  <div class="unresolved-title">Unknown QR Code</div>
  <pre class="raw-qr" id="raw-qr"></pre>
  <button class="submit-btn" style="background:#2a2e3e;color:#8b90a0" onclick="cancelResult()">Scan Again</button>
</div>

<div class="toast" id="toast">Updated ✓</div>

<script>
(function(){
  const API = '/phone/%s/api';
  let mode = 'delta';
  let currentComp = null;
  let scanning = false;
  let stream = null;
  let detector = null;
  let animId = null;
  let usingFallback = false;
  let fallbackCanvas = null;
  let fallbackCtx = null;

  const $ = id => document.getElementById(id);

  // ---- Mode toggle ----
  window.setMode = function(m) {
    mode = m;
    $('mode-delta').classList.toggle('active', m === 'delta');
    $('mode-set').classList.toggle('active', m === 'set');
    $('quick-btns').style.display = m === 'delta' ? 'flex' : 'none';
    $('qty-label').textContent = m === 'delta' ? 'Add to current count' : 'Set exact count';
    $('qty-input').value = '0';
  };

  // ---- Quick add ----
  window.quickAdd = function(n) {
    const inp = $('qty-input');
    inp.value = String((parseInt(inp.value,10)||0) + n);
  };

  // ---- Show/hide sheets ----
  function showSheet(id) {
    $(id).classList.add('visible');
  }
  function hideSheets() {
    $('result-sheet').classList.remove('visible');
    $('unresolved-sheet').classList.remove('visible');
  }

  function showToast(msg) {
    const t = $('toast');
    t.textContent = msg || 'Updated ✓';
    t.classList.add('visible');
    setTimeout(() => t.classList.remove('visible'), 1600);
  }

  // ---- Scanner ----
  function loadJsQR() {
    if (typeof jsQR !== 'undefined') return Promise.resolve(true);
    return new Promise(resolve => {
      const s = document.createElement('script');
      s.src = 'https://cdn.jsdelivr.net/npm/jsqr@1/dist/jsQR.js';
      s.onload = () => resolve(true);
      s.onerror = () => resolve(false);
      document.head.appendChild(s);
    });
  }

  async function startScanner() {
    const label = $('scan-label');
    if (!('BarcodeDetector' in window)) {
      label.textContent = 'Loading scanner…';
      const loaded = await loadJsQR();
      if (!loaded) {
        label.textContent = 'Scanner unavailable — check your internet connection';
        label.classList.add('error');
        return;
      }
      usingFallback = true;
    } else {
      detector = new BarcodeDetector({formats:['qr_code','ean_13','ean_8','code_128','code_39']});
      usingFallback = false;
    }
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        video:{facingMode:'environment',width:{ideal:1280},height:{ideal:720}}
      });
    } catch(e) {
      label.textContent = 'Camera access denied';
      label.classList.add('error');
      return;
    }
    const video = $('camera');
    video.srcObject = stream;
    await video.play();
    scanning = true;
    label.textContent = 'Point camera at a QR code';
    label.classList.remove('error');
    $('scanner-view').classList.remove('scanner-paused');
    scanLoop();
  }

  function scanLoop() {
    if(!scanning) return;
    const video = $('camera');
    if (usingFallback) {
      if (video.readyState === video.HAVE_ENOUGH_DATA) {
        if (!fallbackCanvas) {
          fallbackCanvas = document.createElement('canvas');
          fallbackCtx = fallbackCanvas.getContext('2d');
        }
        fallbackCanvas.width = video.videoWidth;
        fallbackCanvas.height = video.videoHeight;
        fallbackCtx.drawImage(video, 0, 0);
        const imageData = fallbackCtx.getImageData(0, 0, fallbackCanvas.width, fallbackCanvas.height);
        const code = jsQR(imageData.data, imageData.width, imageData.height, {inversionAttempts:'dontInvert'});
        if (code) {
          scanning = false;
          $('scanner-view').classList.add('scanner-paused');
          onDetected(code.data);
          return;
        }
      }
      animId = requestAnimationFrame(scanLoop);
    } else {
      detector.detect(video).then(codes => {
        if(!scanning) return;
        if(codes.length > 0) {
          scanning = false;
          $('scanner-view').classList.add('scanner-paused');
          onDetected(codes[0].rawValue);
        } else {
          animId = requestAnimationFrame(scanLoop);
        }
      }).catch(() => {
        if(scanning) animId = requestAnimationFrame(scanLoop);
      });
    }
  }

  function resumeScanner() {
    hideSheets();
    mode = 'delta';
    $('mode-delta').classList.add('active');
    $('mode-set').classList.remove('active');
    $('quick-btns').style.display = 'flex';
    $('qty-label').textContent = 'Add to current count';
    $('qty-input').value = '0';
    currentComp = null;
    scanning = true;
    $('scanner-view').classList.remove('scanner-paused');
    $('scan-label').textContent = 'Point camera at a QR code';
    scanLoop();
  }

  // ---- Lookup ----
  async function onDetected(qrData) {
    $('scan-label').textContent = 'Looking up…';
    try {
      const res = await fetch(API + '/lookup', {
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({qrData})
      });
      const data = await res.json();
      if(data.found) {
        showResult(data);
      } else {
        showUnresolved(data.rawQr || qrData);
      }
    } catch(e) {
      showUnresolved(qrData);
    }
  }

  function showResult(data) {
    currentComp = data;
    // Image
    const img = $('comp-image');
    if(data.imageUrl) {
      img.src = data.imageUrl;
      img.classList.remove('hidden');
    } else {
      img.classList.add('hidden');
    }
    // Title
    $('comp-title').textContent = data.displayName || 'Component';
    let sub = [];
    if(data.description) sub.push(data.description);
    $('comp-sub').textContent = sub.join(' · ');

    // Info grid
    const grid = $('info-grid');
    grid.innerHTML = '';
    const fields = [
      ['Manufacturer', data.manufacturer],
      ['MPN', data.mpn],
      ['Package', data.package],
      ['Current Qty', data.quantity != null ? data.quantity : '—'],
      ['Location', data.location],
      ['Bag', data.bagLabel],
    ];
    for(const [label,val] of fields) {
      if(!val && val !== 0) continue;
      grid.innerHTML += '<div class="info-label">' + esc(label) + '</div><div class="info-value">' + esc(String(val)) + '</div>';
    }

    setMode('delta');
    showSheet('result-sheet');
  }

  function showUnresolved(raw) {
    $('raw-qr').textContent = raw;
    showSheet('unresolved-sheet');
  }

  // ---- Submit ----
  window.submitCount = async function() {
    if(!currentComp) return;
    const btn = $('submit-btn');
    btn.disabled = true;
    btn.textContent = 'Updating…';
    const val = parseInt($('qty-input').value, 10) || 0;
    try {
      const res = await fetch(API + '/submit', {
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({
          componentId: currentComp.componentId,
          mode: mode,
          value: val
        })
      });
      const data = await res.json();
      if(data.success) {
        showToast('Updated — Qty: ' + (data.quantity != null ? data.quantity : '?'));
        setTimeout(resumeScanner, 1200);
      } else {
        showToast(data.error || 'Update failed');
        btn.disabled = false;
        btn.textContent = 'Submit';
      }
    } catch(e) {
      showToast('Network error');
      btn.disabled = false;
      btn.textContent = 'Submit';
    }
  };

  window.cancelResult = function() {
    resumeScanner();
  };

  function esc(s){
    const d=document.createElement('div');d.textContent=s;return d.innerHTML;
  }

  // ---- Init ----
  startScanner();
})();
</script>
</body>
</html>`, token)
}
