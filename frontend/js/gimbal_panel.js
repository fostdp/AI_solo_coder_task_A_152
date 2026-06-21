export class GimbalPanel {
    constructor(apiBase, wsUrl, threeDRenderer) {
        this.API_BASE = apiBase;
        this.WS_URL = wsUrl;
        this.threeDRenderer = threeDRenderer;

        this.censers = [];
        this.currentCenser = null;
        this.chartData = {
            innerAngles: [],
            outerAngles: [],
            tilts: [],
            balanceScores: [],
            spillRisks: [],
            times: []
        };
        this.MAX_CHART_POINTS = 60;

        this.ws = null;
        this.reconnectDelay = 3000;
        this.maxReconnectDelay = 60000;

        this.init();
    }

    async init() {
        this.loadCensers();
        this.connectWebSocket();
        this.setupEventListeners();
        this.startClock();
        this.startChartRedraw();
    }

    async loadCensers() {
        try {
            const res = await fetch(`${this.API_BASE}/censers`);
            this.censers = await res.json();
            const select = document.getElementById('censer-select');
            if (select) {
                select.innerHTML = this.censers.map(c =>
                    `<option value="${c.id}">${c.code} - ${c.name}</option>`
                ).join('');
            }
            if (this.censers.length > 0) {
                this.selectCenser(this.censers[0].id);
            }
        } catch (e) {
            console.error('Failed to load censers:', e);
        }
    }

    async selectCenser(id) {
        this.currentCenser = this.censers.find(c => c.id === id);
        try {
            const res = await fetch(`${this.API_BASE}/censers/${id}/config`);
            const config = await res.json();
            this.updateConfigDisplay(config);
        } catch (e) {
            console.error('Failed to load config:', e);
        }
        this.chartData = { innerAngles: [], outerAngles: [], tilts: [], balanceScores: [], spillRisks: [], times: [] };
        this.loadLatestData(id);
    }

    updateConfigDisplay(cfg) {
        const el = (id) => document.getElementById(id);
        if (el('cfg-inner-mass')) el('cfg-inner-mass').textContent = cfg.inner_ring_mass.toFixed(3) + ' kg';
        if (el('cfg-outer-mass')) el('cfg-outer-mass').textContent = cfg.outer_ring_mass.toFixed(3) + ' kg';
        if (el('cfg-body-mass')) el('cfg-body-mass').textContent = cfg.body_mass.toFixed(3) + ' kg';
        if (el('cfg-damping')) el('cfg-damping').textContent = cfg.damping_coefficient.toFixed(3);
        if (el('cfg-friction')) el('cfg-friction').textContent = cfg.friction_coefficient.toFixed(3);
        if (el('cfg-tilt-th')) el('cfg-tilt-th').textContent = cfg.tilt_alarm_threshold.toFixed(1) + '°';
        const viscosity = cfg.perfume_viscosity != null ? cfg.perfume_viscosity : 0.5;
        const fillRatio = cfg.fill_ratio != null ? cfg.fill_ratio : 0.6;
        if (el('cfg-viscosity')) el('cfg-viscosity').textContent = viscosity.toFixed(3) + ' Pa·s';
        if (el('cfg-fill-ratio')) el('cfg-fill-ratio').textContent = (fillRatio * 100).toFixed(0) + '%';
    }

    async loadLatestData(censerId) {
        try {
            const res = await fetch(`${this.API_BASE}/censers/${censerId}/sensor-data?limit=60`);
            const data = await res.json();
            data.reverse().forEach(d => this.pushChartData(d));
            if (data.length > 0) {
                const latest = data[data.length - 1];
                this.updateMetrics(latest);
                this.threeDRenderer.updateGimbalAngles(
                    latest.inner_ring_angle || 0,
                    latest.outer_ring_angle || 0,
                    latest.body_tilt || 0
                );
            }
            this.drawAllCharts();
        } catch (e) {
            console.error('Failed to load sensor data:', e);
        }
    }

    pushChartData(d) {
        const now = d.time ? new Date(d.time) : new Date();
        this.chartData.times.push(now);
        this.chartData.innerAngles.push(d.inner_ring_angle || 0);
        this.chartData.outerAngles.push(d.outer_ring_angle || 0);
        this.chartData.tilts.push(d.body_tilt || 0);
        this.chartData.balanceScores.push(d.balance_score != null ? d.balance_score : 1);
        this.chartData.spillRisks.push(d.spill_risk != null ? d.spill_risk : 0);

        if (this.chartData.times.length > this.MAX_CHART_POINTS) {
            this.chartData.times.shift();
            this.chartData.innerAngles.shift();
            this.chartData.outerAngles.shift();
            this.chartData.tilts.shift();
            this.chartData.balanceScores.shift();
            this.chartData.spillRisks.shift();
        }
    }

    updateMetrics(d) {
        const el = (id) => document.getElementById(id);
        if (el('ov-inner')) el('ov-inner').textContent = (d.inner_ring_angle || 0).toFixed(2) + '°';
        if (el('ov-outer')) el('ov-outer').textContent = (d.outer_ring_angle || 0).toFixed(2) + '°';
        if (el('ov-tilt')) el('ov-tilt').textContent = (d.body_tilt || 0).toFixed(2) + '°';
        if (el('ov-slosh')) el('ov-slosh').textContent = (d.slosh_acceleration || 0).toFixed(2) + ' m/s²';

        if (el('body-tilt')) el('body-tilt').textContent = (d.body_tilt || 0).toFixed(2) + '°';
        if (el('tilt-bar')) el('tilt-bar').style.width = Math.min(100, (d.body_tilt || 0) * 4) + '%';

        const balance = d.balance_score != null ? d.balance_score : 1;
        const balanceEl = el('balance-score');
        const balanceBar = el('balance-bar');
        if (balanceEl) balanceEl.textContent = (balance * 100).toFixed(1) + '%';
        if (balanceBar) balanceBar.style.width = (balance * 100) + '%';
        if (balanceEl) balanceEl.className = 'metric-value ' + this.getColorClass(balance, 0.7, 0.4);
        if (balanceBar) balanceBar.className = 'progress-fill ' + this.getBarClass(balance, 0.7, 0.4, true);

        const spill = d.spill_risk != null ? d.spill_risk : 0;
        const spillEl = el('spill-risk');
        const spillBar = el('spill-bar');
        if (spillEl) spillEl.textContent = (spill * 100).toFixed(1) + '%';
        if (spillBar) spillBar.style.width = (spill * 100) + '%';
        if (spillEl) spillEl.className = 'metric-value ' + this.getColorClass(1 - spill, 0.6, 0.3);
        if (spillBar) spillBar.className = 'progress-fill ' + this.getBarClass(spill, 0.3, 0.6, false);
    }

    getColorClass(value, warnThresh, critThresh) {
        if (value > warnThresh) return 'green';
        if (value > critThresh) return 'yellow';
        return 'red';
    }

    getBarClass(value, warnThresh, critThresh, invert) {
        const v = invert ? 1 - value : value;
        if (v < warnThresh) return 'green';
        if (v < critThresh) return 'yellow';
        return 'red';
    }

    drawAllCharts() {
        this.drawLineChart('chart-rings', [
            { data: this.chartData.innerAngles, color: '#a78bfa', label: '内环' },
            { data: this.chartData.outerAngles, color: '#22d3ee', label: '外环' }
        ], -45, 45);
        this.drawLineChart('chart-tilt', [
            { data: this.chartData.tilts, color: '#22d3ee', label: '倾角' }
        ], 0, 30);
        this.drawLineChart('chart-balance', [
            { data: this.chartData.balanceScores, color: '#4ade80', label: '平衡' },
            { data: this.chartData.spillRisks, color: '#f87171', label: '风险' }
        ], 0, 1);
    }

    drawLineChart(canvasId, series, yMin, yMax) {
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

    connectWebSocket() {
        const dot = document.getElementById('conn-dot');
        const status = document.getElementById('conn-status');

        try {
            this.ws = new WebSocket(this.WS_URL);

            this.ws.onopen = () => {
                if (dot) {
                    dot.style.background = '#4ade80';
                    dot.style.boxShadow = '0 0 8px #4ade80';
                }
                if (status) status.textContent = '实时连接';
                this.reconnectDelay = 3000;
            };

            this.ws.onmessage = (event) => {
                try {
                    const msg = JSON.parse(event.data);
                    if (msg.type === 'sensor_data') {
                        if (this.currentCenser && msg.data.censer_id === this.currentCenser.id) {
                            const d = msg.data;
                            this.pushChartData(d);
                            this.updateMetrics(d);
                            this.threeDRenderer.updateGimbalAngles(
                                d.inner_ring_angle || 0,
                                d.outer_ring_angle || 0,
                                d.body_tilt || 0
                            );
                            this.drawAllCharts();
                        }
                    } else if (msg.type === 'alert') {
                        this.showAlert(msg.data);
                    }
                } catch (e) {
                    console.error('Parse WS message failed:', e);
                }
            };

            this.ws.onclose = () => {
                if (dot) {
                    dot.style.background = '#f87171';
                    dot.style.boxShadow = '0 0 8px #f87171';
                }
                if (status) status.textContent = '已断开';

                setTimeout(() => {
                    this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
                    this.connectWebSocket();
                }, this.reconnectDelay);
            };

            this.ws.onerror = () => {
                if (this.ws) this.ws.close();
            };
        } catch (e) {
            console.error('WS connection failed:', e);
        }
    }

    showAlert(alert) {
        const feed = document.getElementById('alert-feed');
        if (!feed) return;
        const item = document.createElement('div');
        item.className = 'alert-item ' + (alert.severity || 'warning');
        const time = new Date(alert.created_at || Date.now()).toLocaleTimeString();
        item.textContent = `[${time}] ${(alert.severity || 'warning').toUpperCase()}: ${alert.message}`;
        feed.insertBefore(item, feed.firstChild);

        while (feed.children.length > 5) {
            feed.removeChild(feed.lastChild);
        }

        setTimeout(() => {
            if (item.parentNode) item.parentNode.removeChild(item);
        }, 8000);
    }

    async runSloshAnalysis(motionType) {
        if (!this.currentCenser) return;

        const resultEl = document.getElementById('analysis-result');
        if (resultEl) resultEl.innerHTML = '<div style="color:#d4a853;">分析中...</div>';

        try {
            const res = await fetch(`${this.API_BASE}/censers/${this.currentCenser.id}/slosh-analysis`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ motion_type: motionType })
            });
            const result = await res.json();

            const spillColor = result.spill_probability > 0.5 ? '#f87171' :
                              result.spill_probability > 0.2 ? '#fbbf24' : '#4ade80';

            if (resultEl) {
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
            }
        } catch (e) {
            if (resultEl) resultEl.innerHTML = '<div style="color:#f87171;">分析失败</div>';
            console.error('Analysis failed:', e);
        }
    }

    startClock() {
        const updateClock = () => {
            const el = document.getElementById('current-time');
            if (el) el.textContent = new Date().toLocaleTimeString();
        };
        updateClock();
        setInterval(updateClock, 1000);
    }

    startChartRedraw() {
        setInterval(() => {
            if (this.chartData.times.length > 0) this.drawAllCharts();
        }, 500);
    }

    setupEventListeners() {
        const censerSelect = document.getElementById('censer-select');
        if (censerSelect) {
            censerSelect.addEventListener('change', (e) => {
                this.selectCenser(e.target.value);
            });
        }

        document.querySelectorAll('.analysis-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('.analysis-btn').forEach(b => b.classList.remove('active'));
                e.target.classList.add('active');
                this.runSloshAnalysis(e.target.dataset.motion);
            });
        });
    }
}
