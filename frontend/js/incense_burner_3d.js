import * as THREE from 'three';

export class IncenseBurner3D {
    constructor(canvasElement) {
        this.canvas = canvasElement;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.outerRing = null;
        this.innerRing = null;
        this.censerBody = null;

        this.cameraAngleX = 0.7;
        this.cameraAngleY = 0.55;
        this.cameraDistance = 5.5;

        this.isDragging = false;
        this.prevMouseX = 0;
        this.prevMouseY = 0;

        this.init();
    }

    init() {
        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x050810);
        this.scene.fog = new THREE.Fog(0x050810, 8, 20);

        const container = this.canvas;
        this.camera = new THREE.PerspectiveCamera(45, container.clientWidth / container.clientHeight, 0.1, 100);
        this.camera.position.set(3.5, 2.5, 4);
        this.camera.lookAt(0, 0, 0);

        this.renderer = new THREE.WebGLRenderer({ canvas: container, antialias: true });
        this.renderer.setSize(container.clientWidth, container.clientHeight);
        this.renderer.setPixelRatio(window.devicePixelRatio);

        this.addLights();
        this.createGroundGrid();
        this.createCenserModel();
        this.setupInteraction();

        window.addEventListener('resize', () => this.handleResize());

        this.animate();
    }

    addLights() {
        const ambientLight = new THREE.AmbientLight(0x404060, 0.6);
        this.scene.add(ambientLight);

        const keyLight = new THREE.DirectionalLight(0xffeedd, 0.9);
        keyLight.position.set(5, 8, 5);
        this.scene.add(keyLight);

        const fillLight = new THREE.DirectionalLight(0xd4a853, 0.4);
        fillLight.position.set(-5, 3, -5);
        this.scene.add(fillLight);

        const rimLight = new THREE.PointLight(0xd4a853, 0.8, 10);
        rimLight.position.set(0, -3, 0);
        this.scene.add(rimLight);
    }

    createGroundGrid() {
        const gridHelper = new THREE.GridHelper(10, 20, 0x1a2233, 0x121826);
        gridHelper.position.y = -2.5;
        this.scene.add(gridHelper);

        const groundGeometry = new THREE.CircleGeometry(5, 64);
        const groundMaterial = new THREE.MeshBasicMaterial({
            color: 0x0a0e17,
            transparent: true,
            opacity: 0.6
        });
        const ground = new THREE.Mesh(groundGeometry, groundMaterial);
        ground.rotation.x = -Math.PI / 2;
        ground.position.y = -2.49;
        this.scene.add(ground);
    }

    createCenserModel() {
        const outerGroup = new THREE.Group();
        outerGroup.renderOrder = 1;
        this.scene.add(outerGroup);

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

        this.addDecorativePattern(bodyGroup);

        this.outerRing = outerGroup;
        this.innerRing = innerGroup;
        this.censerBody = bodyGroup;
    }

    addDecorativePattern(group) {
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

    setupInteraction() {
        const container = this.canvas;

        container.addEventListener('mousedown', (e) => {
            this.isDragging = true;
            this.prevMouseX = e.clientX;
            this.prevMouseY = e.clientY;
        });

        container.addEventListener('mousemove', (e) => {
            if (!this.isDragging) return;
            const dx = e.clientX - this.prevMouseX;
            const dy = e.clientY - this.prevMouseY;
            this.cameraAngleX -= dx * 0.008;
            this.cameraAngleY = Math.max(0.1, Math.min(Math.PI / 2 - 0.1, this.cameraAngleY - dy * 0.008));
            this.prevMouseX = e.clientX;
            this.prevMouseY = e.clientY;
        });

        container.addEventListener('mouseup', () => this.isDragging = false);
        container.addEventListener('mouseleave', () => this.isDragging = false);

        container.addEventListener('wheel', (e) => {
            e.preventDefault();
            this.cameraDistance = Math.max(3, Math.min(12, this.cameraDistance + e.deltaY * 0.005));
        });
    }

    updateCameraPosition() {
        this.camera.position.x = this.cameraDistance * Math.sin(this.cameraAngleX) * Math.cos(this.cameraAngleY);
        this.camera.position.y = this.cameraDistance * Math.sin(this.cameraAngleY);
        this.camera.position.z = this.cameraDistance * Math.cos(this.cameraAngleX) * Math.cos(this.cameraAngleY);
        this.camera.lookAt(0, 0, 0);
    }

    updateGimbalAngles(innerAngle, outerAngle, bodyTilt) {
        if (this.outerRing) {
            this.outerRing.rotation.z = THREE.MathUtils.degToRad(outerAngle);
        }
        if (this.innerRing) {
            this.innerRing.rotation.x = THREE.MathUtils.degToRad(innerAngle);
        }
        if (this.censerBody) {
            this.censerBody.rotation.y = THREE.MathUtils.degToRad(bodyTilt * 0.5);
        }
    }

    handleResize() {
        const container = this.canvas;
        this.camera.aspect = container.clientWidth / container.clientHeight;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(container.clientWidth, container.clientHeight);
    }

    animate() {
        requestAnimationFrame(() => this.animate());
        this.updateCameraPosition();
        this.renderer.render(this.scene, this.camera);
    }
}
