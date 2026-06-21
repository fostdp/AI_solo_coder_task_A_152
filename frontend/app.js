import { IncenseBurner3D } from './js/incense_burner_3d.js';
import { GimbalPanel } from './js/gimbal_panel.js';

const API_BASE = 'http://localhost:8080/api/v1';
const WS_URL = 'ws://localhost:8080/ws';

let threeDRenderer = null;
let gimbalPanel = null;

document.addEventListener('DOMContentLoaded', () => {
    const canvas = document.getElementById('three-canvas');
    if (canvas) {
        threeDRenderer = new IncenseBurner3D(canvas);
    }

    if (threeDRenderer) {
        gimbalPanel = new GimbalPanel(API_BASE, WS_URL, threeDRenderer);
    }

    console.log('Censer Simulation System initialized');
});
