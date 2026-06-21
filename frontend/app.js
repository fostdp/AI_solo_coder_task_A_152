import * as THREE from 'three';

const API_BASE = 'http://localhost:8080/api/v1';
const WS_URL = 'ws://localhost:8080/ws';

let scene, camera, renderer;
let outerRing, innerRing, censerBody;
let censers = [];
let currentCenser = null;
let chartData = {
    innerAngles: [],
    outerAngles: [],
    tilts: [],
    balanceScores: [],
    spillRisks: [],
    times: []
};
const MAX_CHART_POINTS = 60;

function initThreeJS() {
    const container = document.getElementById('three-canvas');
    scene = new THREE.Scene();
    scene.background = new THREE.Color(0x050810);
    scene.fog = new THREE.Fog(0x050810, 8, 20);

    camera = new THREE.PerspectiveCamera(45, container.clientWidth / container.clientHeight, 0.1, 100);
    camera.position.set(3.5, 2.5, 4);
    camera.lookAt(0, 0, 0);

    renderer = new THREE.WebGLRenderer({ canvas: container, antialias: true });
    renderer.setSize(container.clientWidth, container.clientHeight);
    renderer.setPixelRatio(window.devicePixelRatio);

    const ambientLight = new THREE.AmbientLight(0x404060, 0.6);
    scene.add(ambientLight);

    const keyLight = new THREE.DirectionalLight(0xffeedd, 0.9);
    keyLight.position.set(5, 8, 5);
    scene.add(keyLight);

    const fillLight = new THREE.DirectionalLight(0xd4a853, 0.4);
    fillLight.position.set(-5, 3, -5);
    scene.add(fillLight);

    const rimLight = new THREE.PointLight(0xd4a853, 0.8, 10);
    rimLight.position.set(0, -3, 0);
    scene.add(rimLight);

    createGroundGrid();
    createCenserModel();

    let isDragging = false;
    let prevMouseX = 0, prevMouseY = 0;
    let cameraAngleX = 0.7, cameraAngleY = 0.55;
    let cameraDistance = 5.5;

    container.addEventListener('mousedown', (e) => {
        isDragging = true;
        prevMouseX = e.clientX;
        prevMouseY = e.clientY;
    });

    container.addEventListener('mousemove', (e) => {
        if (!isDragging) return;
        const dx = e.clientX - prevMouseX;
        const dy = e.clientY - prevMouseY;
        cameraAngleX -= dx * 0.008;
        cameraAngleY = Math.max(0.1, Math.min(Math.PI / 2 - 0.1, cameraAngleY - dy * 0.008));
        prevMouseX = e.clientX;
        prevMouseY = e.clientY;
    });

    container.addEventListener('mouseup', () => isDragging = false);
    container.addEventListener('mouseleave', () => isDragging = false);

    container.addEventListener('wheel', (e) => {
        e.preventDefault();
        cameraDistance = Math.max(3, Math.min(12, cameraDistance + e.deltaY * 0.005));
    });

    function updateCameraPosition() {
        camera.position.x = cameraDistance * Math.sin(cameraAngleX) * Math.cos(cameraAngleY);
        camera.position.y = cameraDistance * Math.sin(cameraAngleY);
        camera.position.z = cameraDistance * Math.cos(cameraAngleX) * Math.cos(cameraAngleY);
        camera.lookAt(0, 0, 0);
    }

    window.addEventListener('resize', () => {
        camera.aspect = container.clientWidth / container.clientHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(container.clientWidth, container.clientHeight);
    });

    function animate() {
        requestAnimationFrame(animate);
        updateCameraPosition();
        renderer.render(scene, camera);
    }
    animate();
}

function createGroundGrid() {
    const gridHelper = new THREE.GridHelper(10, 20, 0x1a2233, 0x121826);
    gridHelper.position.y = -2.5;
    scene.add(gridHelper);

    const groundGeometry = new THREE.CircleGeometry(5, 64);
    const groundMaterial = new THREE.MeshBasicMaterial({
        color: 0x0a0e17,
        transparent: true,
        opacity: 0.6
    });
    const ground = new THREE.Mesh(groundGeometry, groundMaterial);
    ground.rotation.x = -Math.PI / 2;
    ground.position.y = -2.49;
    scene.add(ground);
}

function createCenserModel() {
    const outerGroup = new THREE.Group();
    outerGroup.renderOrder = 1;
    scene.add(outerGroup);

    const outerRingGeo = new THREE.TorusGeometry(1.5, 0.04, 16, 80);
    const outerWireMat = new THREE.MeshBasicMaterial({
        color: 0x22d3ee,
        wireframe: true,
        transparent: true,
        opacity: 0.7,
        depthWrite: false,
        depthTest: true
    });
    const outerRingMesh = new THREE.Mesh(outerRingGeo, outerWireMat);
    outerRingMesh.renderOrder = 1;
    outerGroup.add(outerRingMesh);

    const outerSolidGeo = new THREE.TorusGeometry(1.5, 0.015, 8, 80);
    const outerSolidMat = new THREE.MeshPhongMaterial({
        color: 0x22d3ee,
        emissive: 0x0891b2,
        emissiveIntensity: 0.3,
        transparent: true,
        opacity: 0.5,
        depthWrite: false,
        depthTest: true
    });
    const outerSolid = new THREE.Mesh(outerSolidGeo, outerSolidMat);
    outerSolid.renderOrder = 1;
    outerGroup.add(outerSolid);

    const innerGroup = new THREE.Group();
    innerGroup.renderOrder = 2;
    outerGroup.add(innerGroup);

    const innerRingGeo = new THREE.TorusGeometry(1.1, 0.035, 16, 70);
    const innerWireMat = new THREE.MeshBasicMaterial({
        color: 0xa78bfa,
        wireframe: true,
        transparent: true,
        opacity: 0.75,
        depthWrite: false,
        depthTest: true
    });
    const innerRingMesh = new THREE.Mesh(innerRingGeo, innerWireMat);
    innerRingMesh.rotation.x = Math.PI / 2;
    innerRingMesh.renderOrder = 2;
    innerGroup.add(innerRingMesh);

    const innerSolidGeo = new THREE.TorusGeometry(1.1, 0.012, 8, 70);
    const innerSolidMat = new THREE.MeshPhongMaterial({
        color: 0xa78bfa,
        emissive: 0x7c3aed,
        emissiveIntensity: 0.3,
        transparent: true,
        opacity: 0.5,
        depthWrite: false,
        depthTest: true
    });
    const innerSolid = new THREE.Mesh(innerSolidGeo, innerSolidMat);
    innerSolid.rotation.x = Math.PI / 2;
    innerSolid.renderOrder = 2;
    innerGroup.add(innerSolid);

    const bodyGroup = new THREE.Group();
    bodyGroup.renderOrder = 3;
    innerGroup.add(bodyGroup);

    const bodyGeo = new THREE.SphereGeometry(0.55, 48, 32);
    const bodyMat = new THREE.MeshPhongMaterial({
        color: 0xd4a853,
        emissive: 0x92400e,
        emissiveIntensity: 0.2,
        shininess: 80,
        specular: 0xf4d487,
        transparent: true,
        opacity: 0.92,
        depthWrite: true,
        depthTest: true
    });
    const body = new THREE.Mesh(bodyGeo, bodyMat);
    body.renderOrder = 3;
    bodyGroup.add(body);

    const shellGeo = new THREE.SphereGeometry(0.58, 48, 32, 0, Math.PI * 2, 0, Math.PI / 2);
    const shellMat = new THREE.MeshPhongMaterial({
        color: 0xb8860b,
        emissive: 0x78350f,
        emissiveIntensity: 0.15,
        shininess: 100,
        transparent: true,
        opacity: 0.35,
        side: THREE.DoubleSide,
        depthWrite: false,
        depthTest: true
    });
    const shell = new THREE.Mesh(shellGeo, shellMat);
    shell.renderOrder = 4;
    bodyGroup.add(shell);

    const glowGeo = new THREE.SphereGeometry(0.25, 24, 24);
    const glowMat = new THREE.MeshBasicMaterial({
        color: 0xff6600,
        transparent: true,
        opacity: 0.85,
        depthWrite: false,
        depthTest: true
    });
    const glow = new THREE.Mesh(glowGeo, glowMat);
    glow.position.y = -0.1;
    glow.renderOrder = 5;
    bodyGroup.add(glow);

    const light = new THREE.PointLight(0xff5500, 1.5, 4);
    light.position.y = -0.1;
    bodyGroup.add(light);

    addDecorativePattern(bodyGroup);

    outerRing = outerGroup;
    innerRing = innerGroup;
    censerBody = bodyGroup;
}

function addDecorativePattern(group) {
    const patternMat = new THREE.MeshPhongMaterial({
        color: 0xf4d487,
        emissive: 0x92400e,
        emissiveIntensity: 0.3,
        shininess: 120,
        depthWrite: true,
        depthTest: true
    });

    for (let i = 0; i < 12; i++) {
        const angle = (i / 12) * Math.PI * 2;
        const dotGeo = new THREE.SphereGeometry(0.025, 12, 12);
        const dot = new THREE.Mesh(dotGeo, patternMat);
        dot.renderOrder = 4;
        dot.position.set(
            Math.cos(angle) * 0.5,
            0.2,
            Math.sin(angle) * 0.5
        );
        group.add(dot);
    }

    const vineGeo = new THREE.TorusGeometry(0.5, 0.008, 8, 100);
    const vineMat = new THREE.MeshPhongMaterial({
        color: 0xd4a853,
        emissive: 0x78350f,
        emissiveIntensity: 0.2,
        transparent: true,
        opacity: 0.8
    });
    const vine1 = new THREE.Mesh(vineGeo, vineMat);
    vine1.rotation.x = Math.PI / 2;
    vine1.position.y = 0.15;
    group.add(vine1);

    const vine2 = new THREE.Mesh(vineGeo, vineMat);
    vine2.rotation.x = Math.PI / 2;
    vine2.position.y = -0.15;
    group.add(vine2);
}

function updateGimbalAngles(innerAngle, outerAngle, bodyTilt) {
    if (outerRing) {
        outerRing.rotation.z = THREE.MathUtils.degToRad(outerAngle);
    }
    if (innerRing) {
        innerRing.rotation.x = THREE.MathUtils.degToRad(innerAngle);
    }
    if (censerBody) {
        censerBody.rotation.y = THREE.MathUtils.degToRad(bodyTilt * 0.5);
    }
}

async function loadCensers() {
    try {
        const res = await fetch(`${API_BASE}/censers`);
        censers = await res.json();
        const select = document.getElementById('censer-select');
        select.innerHTML = censers.map(c =>
            `<option value="${c.id}">${c.code} - ${c.name}</option>`
        ).join('');
        if (censers.length > 0) {
            currentCenser = censers[0];
            selectCenser(censers[0].id);
        }
    } catch (e) {
        console.error('Failed to load censers:', e);
    }
}

async function selectCenser(id) {
    currentCenser = censers.find(c => c.id === id);
    try {
        const res = await fetch(`${API_BASE}/censers/${id}/config`);
        const config = await res.json();
        updateConfigDisplay(config);
    } catch (e) {
        console.error('Failed to load config:', e);
    }
    chartData = { innerAngles: [], outerAngles: [], tilts: [], balanceScores: [], spillRisks: [], times: [] };
    loadLatestData(id);
}

function updateConfigDisplay(cfg) {
    document.getElementById('cfg-inner-mass').textContent = cfg.inner_ring_mass.toFixed(3) + ' kg';
    document.getElementById('cfg-outer-mass').textContent = cfg.outer_ring_mass.toFixed(3) + ' kg';
    document.getElementById('cfg-body-mass').textContent = cfg.body_mass.toFixed(3) + ' kg';
    document.getElementById('cfg-damping').textContent = cfg.damping_coefficient.toFixed(3);
    document.getElementById('cfg-friction').textContent = cfg.friction_coefficient.toFixed(3);
    document.getElementById('cfg-tilt-th').textContent = cfg.tilt_alarm_threshold.toFixed(1) + '°';
    const viscosity = cfg.perfume_viscosity != null ? cfg.perfume_viscosity : 0.5;
    const fillRatio = cfg.fill_ratio != null ? cfg.fill_ratio : 0.6;
    document.getElementById('cfg-viscosity').textContent = viscosity.toFixed(3) + ' Pa·s';
    document.getElementById('cfg-fill-ratio').textContent = (fillRatio * 100).toFixed(0) + '%';
}

async function loadLatestData(censerId) {
    try {
        const res = await fetch(`${API_BASE}/censers/${censerId}/sensor-data?limit=60`);
        const data = await res.json();
        data.reverse().forEach(d => {
            pushChartData(d);
        });
        if (data.length > 0) {
            const latest = data[data.length - 1];
            updateMetrics(latest);
            updateGimbalAngles(
                latest.inner_ring_angle || 0,
                latest.outer_ring_angle || 0,
                latest.body_tilt || 0
            );
        }
        drawAllCharts();
    } catch (e) {
        console.error('Failed to load sensor data:', e);
    }
}

function pushChartData(d) {
    const now = d.time ? new Date(d.time) : new Date();
    chartData.times.push(now);
    chartData.innerAngles.push(d.inner_ring_angle || 0);
    chartData.outerAngles.push(d.outer_ring_angle || 0);
    chartData.tilts.push(d.body_tilt || 0);
    chartData.balanceScores.push(d.balance_score != null ? d.balance_score : 1);
    chartData.spillRisks.push(d.spill_risk != null ? d.spill_risk : 0);

    if (chartData.times.length > MAX_CHART_POINTS) {
        chartData.times.shift();
        chartData.innerAngles.shift();
        chartData.outerAngles.shift();
        chartData.tilts.shift();
        chartData.balanceScores.shift();
        chartData.spillRisks.shift();
    }
}

function updateMetrics(d) {
    document.getElementById('ov-inner').textContent = (d.inner_ring_angle || 0).toFixed(2) + '°';
    document.getElementById('ov-outer').textContent = (d.outer_ring_angle || 0).toFixed(2) + '°';
    document.getElementById('ov-tilt').textContent = (d.body_tilt || 0).toFixed(2) + '°';
    document.getElementById('ov-slosh').textContent = (d.slosh_acceleration || 0).toFixed(2) + ' m/s²';

    document.getElementById('body-tilt').textContent = (d.body_tilt || 0).toFixed(2) + '°';
    document.getElementById('tilt-bar').style.width = Math.min(100, (d.body_tilt || 0) * 4) + '%';

    const balance = d.balance_score != null ? d.balance_score : 1;
    const balanceEl = document.getElementById('balance-score');
    const balanceBar = document.getElementById('balance-bar');
    balanceEl.textContent = (balance * 100).toFixed(1) + '%';
    balanceBar.style.width = (balance * 100) + '%';
    balanceEl.className = 'metric-value ' + getColorClass(balance, 0.7, 0.4);
    balanceBar.className = 'progress-fill ' + getBarClass(balance, 0.7, 0.4, true);

    const spill = d.spill_risk != null ? d.spill_risk : 0;
    const spillEl = document.getElementById('spill-risk');
    const spillBar = document.getElementById('spill-bar');
    spillEl.textContent = (spill * 100).toFixed(1) + '%';
    spillBar.style.width = (spill * 100) + '%';
    spillEl.className = 'metric-value ' + getColorClass(1 - spill, 0.6, 0.3);
    spillBar.className = 'progress-fill ' + getBarClass(spill, 0.3, 0.6, false);
}

function getColorClass(value, warnThresh, critThresh) {
    if (value > warnThresh) return 'green';
    if (value > critThresh) return 'yellow';
    return 'red';
}

function getBarClass(value, warnThresh, critThresh, invert) {
    const v = invert ? 1 - value : value;
    if (v < warnThresh) return 'green';
    if (v < critThresh) return 'yellow';
    return 'red';
}

function drawAllCharts() {
    drawLineChart('chart-rings', [
        { data: chartData.innerAngles, color: '#a78bfa', label: '内环' },
        { data: chartData.outerAngles, color: '#22d3ee', label: '外环' }
    ], -45, 45);
    drawLineChart('chart-tilt', [
        { data: chartData.tilts, color: '#22d3ee', label: '倾角' }
    ], 0, 30);
    drawLineChart('chart-balance', [
        { data: chartData.balanceScores, color: '#4ade80', label: '平衡' },
        { data: chartData.spillRisks, color: '#f87171', label: '风险' }
    ], 0, 1);
}

function drawLineChart(canvasId, series, yMin, yMax) {
    const canvas = document.getElementById(canvasId);
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const dpr = window.devicePixelRatio || 1;
    const rect = canvas.getBoundingClientRect();

    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    ctx.scale(dpr, dpr);

    const W = rect.width;
    const H = rect.height;
    const padL = 35, padR = 10, padT = 5, padB = 18;
    const chartW = W - padL - padR;
    const chartH = H - padT - padB;

    ctx.clearRect(0, 0, W, H);

    ctx.strokeStyle = '#1a2233';
    ctx.lineWidth = 1;
    for (let i = 0; i <= 4; i++) {
        const y = padT + (chartH / 4) * i;
        ctx.beginPath();
        ctx.moveTo(padL, y);
        ctx.lineTo(W - padR, y);
        ctx.stroke();
    }

    ctx.fillStyle = '#8899aa';
    ctx.font = '9px monospace';
    ctx.textAlign = 'right';
    for (let i = 0; i <= 4; i++) {
        const val = yMax - ((yMax - yMin) / 4) * i;
        const y = padT + (chartH / 4) * i;
        ctx.fillText(val.toFixed(yMax <= 1 ? 1 : 0), padL - 4, y + 3);
    }

    series.forEach(s => {
        if (s.data.length < 2) return;
        ctx.strokeStyle = s.color;
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        s.data.forEach((val, i) => {
            const x = padL + (chartW / (s.data.length - 1)) * i;
            const y = padT + chartH - ((val - yMin) / (yMax - yMin)) * chartH;
            if (i === 0) ctx.moveTo(x, y);
            else ctx.lineTo(x, y);
        });
        ctx.stroke();

        const lastX = padL + chartW;
        const lastVal = s.data[s.data.length - 1];
        const lastY = padT + chartH - ((lastVal - yMin) / (yMax - yMin)) * chartH;
        ctx.fillStyle = s.color;
        ctx.beginPath();
        ctx.arc(lastX, lastY, 3, 0, Math.PI * 2);
        ctx.fill();
    });
}

function connectWebSocket() {
    const dot = document.getElementById('conn-dot');
    const status = document.getElementById('conn-status');

    try {
        const ws = new WebSocket(WS_URL);

        ws.onopen = () => {
            dot.style.background = '#4ade80';
            dot.style.boxShadow = '0 0 8px #4ade80';
            status.textContent = '实时连接';
        };

        ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === 'sensor_data') {
                    if (currentCenser && msg.data.censer_id === currentCenser.id) {
                        const d = msg.data;
                        pushChartData(d);
                        updateMetrics(d);
                        updateGimbalAngles(
                            d.inner_ring_angle || 0,
                            d.outer_ring_angle || 0,
                            d.body_tilt || 0
                        );
                        drawAllCharts();
                    }
                } else if (msg.type === 'alert') {
                    showAlert(msg.data);
                }
            } catch (e) {
                console.error('Parse WS message failed:', e);
            }
        };

        ws.onclose = () => {
            dot.style.background = '#f87171';
            dot.style.boxShadow = '0 0 8px #f87171';
            status.textContent = '已断开';
            setTimeout(connectWebSocket, 3000);
        };

        ws.onerror = () => {
            ws.close();
        };
    } catch (e) {
        console.error('WS connection failed:', e);
    }
}

function showAlert(alert) {
    const feed = document.getElementById('alert-feed');
    const item = document.createElement('div');
    item.className = 'alert-item ' + (alert.severity || 'warning');
    const time = new Date(alert.created_at || Date.now()).toLocaleTimeString();
    item.textContent = `[${time}] ${alert.severity.toUpperCase()}: ${alert.message}`;
    feed.insertBefore(item, feed.firstChild);

    while (feed.children.length > 5) {
        feed.removeChild(feed.lastChild);
    }

    setTimeout(() => {
        if (item.parentNode) item.parentNode.removeChild(item);
    }, 8000);
}

async function runSloshAnalysis(motionType) {
    if (!currentCenser) return;
    document.querySelectorAll('.analysis-btn').forEach(b => b.classList.remove('active'));
    event.target.classList.add('active');

    const resultEl = document.getElementById('analysis-result');
    resultEl.innerHTML = '<div style="color:#d4a853;">分析中...</div>';

    try {
        const res = await fetch(`${API_BASE}/censers/${currentCenser.id}/slosh-analysis`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ motion_type: motionType })
        });
        const result = await res.json();

        const spillColor = result.spill_probability > 0.5 ? '#f87171' :
                          result.spill_probability > 0.2 ? '#fbbf24' : '#4ade80';

        resultEl.innerHTML = `
            <div class="analysis-result-row"><span>运动类型:</span><span style="color:#d4a853;">${result.motion_type}</span></div>
            <div class="analysis-result-row"><span>激励频率:</span><span>${result.frequency.toFixed(2)} Hz</span></div>
            <div class="analysis-result-row"><span>激励振幅:</span><span>${result.amplitude.toFixed(2)} m/s²</span></div>
            <div class="analysis-result-row"><span>阻尼比:</span><span>${result.damping_ratio.toFixed(4)}</span></div>
            <div class="analysis-result-row"><span>共振因子:</span><span>${result.resonance_factor.toFixed(3)}</span></div>
            <div class="analysis-result-row"><span>最大倾角:</span><span>${result.max_tilt_angle.toFixed(2)}°</span></div>
            <div class="analysis-result-row"><span>平衡效率:</span><span>${(result.balance_efficiency * 100).toFixed(1)}%</span></div>
            <div class="analysis-result-row"><span>洒香概率:</span><span style="color:${spillColor};font-weight:bold;">${(result.spill_probability * 100).toFixed(1)}%</span></div>
        `;
    } catch (e) {
        resultEl.innerHTML = '<div style="color:#f87171;">分析失败</div>';
        console.error('Analysis failed:', e);
    }
}

function updateClock() {
    document.getElementById('current-time').textContent = new Date().toLocaleTimeString();
}

document.addEventListener('DOMContentLoaded', () => {
    initThreeJS();
    loadCensers();
    connectWebSocket();
    updateClock();
    setInterval(updateClock, 1000);

    document.getElementById('censer-select').addEventListener('change', (e) => selectCenser(e.target.value));

    document.querySelectorAll('.analysis-btn').forEach(btn => {
        btn.addEventListener('click', (e) => runSloshAnalysis(e.target.dataset.motion));
    });

    setInterval(() => {
        if (chartData.times.length > 0) drawAllCharts();
    }, 500);
});
