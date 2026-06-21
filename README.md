# 古代被中香炉（银熏球）万向平衡机构仿真与抗晃荡分析系统

## 项目概述

本系统为工艺史团队研究唐代被中香炉（银熏球）的复原而开发，实现了：

- **万向平衡机构仿真**：基于多刚体动力学和常平架（Gimbal）理论，计算香炉在任意姿态下的自平衡响应
- **抗晃荡分析**：基于频率响应和阻尼特性，评估香炉在步行、骑马、奔跑、乘车、抬轿等颠簸工况下的洒香概率
- **实时数据采集与告警**：每1分钟通过模拟传感器上报内环转角、外环转角、炉体倾角、晃荡加速度，倾角超限或平衡失效时通过WebSocket推送预警
- **三维可视化**：Three.js绘制香炉三维模型，万向环用透明线框展示，炉体稳定性用动态指标实时展示

---

## 系统架构

```
┌─────────────────┐     HTTP/WS      ┌──────────────────┐    PostgreSQL     ┌───────────────┐
│   前端可视化     │ ◄──────────────► │   Go 后端服务     │ ◄──────────────► │  TimescaleDB  │
│  (Three.js)     │                  │  (Gin + Gorilla) │                  │ (时序数据库)  │
└─────────────────┘                  └──────────────────┘                  └───────────────┘
                                              ▲
                                              │ HTTP POST
                                              │
                                    ┌──────────────────┐
                                    │ 传感器模拟器      │
                                    │ (Python 脚本)     │
                                    └──────────────────┘
```

---

## 目录结构

```
AI_solo_coder_task_A_152/
├── backend/                    # Go 后端服务
│   ├── main.go                 # 主程序入口
│   ├── go.mod                  # Go 模块定义
│   ├── .env.example            # 环境变量示例
│   ├── models/                 # 数据模型定义
│   │   └── models.go
│   ├── database/               # 数据库访问层
│   │   └── database.go
│   ├── simulation/             # 仿真核心算法
│   │   ├── gimbal.go           # 万向平衡机构多刚体动力学
│   │   └── slosh.go            # 抗晃荡频率响应分析
│   ├── alert/                  # 告警管理
│   │   └── alert.go
│   ├── websocket/              # WebSocket Hub
│   │   └── hub.go
│   └── handlers/               # HTTP 处理器
│       └── handlers.go
├── frontend/                   # 前端可视化
│   ├── index.html              # 主页面
│   └── app.js                  # Three.js 3D渲染 + 数据可视化
├── db/                         # 数据库脚本
│   └── init.sql                # TimescaleDB 初始化脚本
└── simulator/                  # 传感器模拟器
    ├── censer_simulator.py     # Python 模拟器脚本
    └── requirements.txt        # Python 依赖
```

---

## 核心算法

### 1. 万向平衡机构动力学模型 (gimbal.go)

基于多刚体动力学方程，模拟三环常平架系统：

```
外环:  I_outer · θ̈_outer = -M_outer·g·R_outer·sin(θ_outer) - τ_accel
内环:  I_inner · θ̈_inner = -M_inner·g·R_inner·sin(θ_inner)·cos(θ_outer) - τ_coupling
炉体:  I_body  · θ̈_body  = -M_body·g_eff·R_body·sin(θ_body) - C_damping·(ω_inner + ω_outer - ω_body)
```

- **I**：转动惯量（圆环 I = mR²，球体 I = 2/5 mR²）
- **C_damping**：阻尼系数，考虑空气阻力和轴承摩擦
- **机械限位**：内外环 ±90° 硬限位，带 50% 反弹系数

### 2. 平衡评分算法

```
平衡评分 = 0.7 · exp(-θ²_tilt / (2·θ²_threshold)) + 0.3 · exp(-|ω_total| / ω_max)
```

综合考虑炉体倾角（高斯衰减）和总角速度，范围 [0, 1]

### 3. 抗晃荡频率响应分析 (slosh.go)

```
共振因子:  Q(ω) = 1 / √[(1 - (ω/ω_n)²)² + (2ζω/ω_n)²]
固有频率:  ω_n = √(g/L_eff)
阻尼比:    ζ = C_actual / C_critical = C·I / (2Mω_n)
```

预设运动模式参数：
| 模式 | 频率(Hz) | 振幅(m/s²) | 说明 |
|------|---------|-----------|------|
| 步行 | 2.0 | 0.5 | 日常行走 |
| 骑马 | 4.0 | 2.0 | 驿骑颠簸 |
| 奔跑 | 6.0 | 1.5 | 急行状态 |
| 乘车 | 8.0 | 1.0 | 马车行驶 |
| 抬轿 | 1.5 | 0.8 | 轿子平稳晃动 |
| 静止 | 0.1 | 0.05 | 静置状态 |

### 4. 洒香概率评估

```
P_spill = 0.5 · f(θ_max) + 0.3 · f(Q) + 0.2 · f(θ_avg)
```

综合最大倾角超限、共振放大、平均倾角三个维度计算。

---

## 快速开始

### 前置依赖

- Go 1.21+
- Python 3.8+
- PostgreSQL 14+  with TimescaleDB 2.x
- 现代浏览器（支持 ES Modules）

---

### 步骤 1：初始化数据库

```bash
# 1. 创建数据库
createdb censer_sim

# 2. 执行初始化脚本
psql -d censer_sim -f db/init.sql
```

脚本会自动创建：
- `censers` 香炉设备表
- `sensor_data` 传感器时序Hypertable
- `alerts` 告警记录表
- `simulation_configs` 仿真参数表
- `slosh_analysis` 抗晃荡分析结果表
- 3 个视图 + 1 个5分钟连续聚合视图
- 3 只示例香炉数据

---

### 步骤 2：启动 Go 后端

```bash
cd backend

# 复制环境变量
cp .env.example .env
# 编辑 .env 修改数据库连接信息

# 下载依赖
go mod tidy

# 运行服务（默认 8080 端口）
go run main.go
```

健康检查：
```bash
curl http://localhost:8080/api/v1/health
```

---

### 步骤 3：启动传感器模拟器

```bash
cd simulator

# 安装依赖
pip install -r requirements.txt

# 基本用法：单设备，步行模式，60秒间隔
python censer_simulator.py

# 快速演示：1秒间隔，骑马模式（会触发告警）
python censer_simulator.py --fast -m horse_riding

# 同时模拟3个香炉
python censer_simulator.py --multi 3 --fast

# 完整参数
python censer_simulator.py \
    -c CENSER-001 \
    -a http://localhost:8080/api/v1 \
    -i 60 \
    -m walking \
    -n 100
```

运行时可输入 `abnormal` 触发异常工况（30秒高振幅），测试告警系统。

---

### 步骤 4：打开前端页面

由于使用 ES Module import，需要通过本地HTTP服务器访问（不能直接双击打开）：

```bash
cd frontend

# 方法1：Python 内置服务器
python -m http.server 8000

# 方法2：Node.js serve
npx serve .
```

浏览器访问 `http://localhost:8000`

---

## API 接口文档

### REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/health` | 服务健康检查 |
| GET | `/api/v1/censers` | 获取所有香炉列表 |
| GET | `/api/v1/censers/:id/config` | 获取指定香炉仿真参数 |
| POST | `/api/v1/sensor-data` | 上报传感器数据 |
| GET | `/api/v1/sensor-data/latest` | 获取所有香炉最新数据 |
| GET | `/api/v1/censers/:id/sensor-data?limit=N` | 获取历史传感器数据 |
| GET | `/api/v1/stability-stats` | 获取1小时稳定性统计 |
| GET | `/api/v1/alerts/active` | 获取未确认告警统计 |
| GET | `/api/v1/censers/:id/alerts` | 获取告警历史 |
| POST | `/api/v1/alerts/:id/acknowledge` | 确认告警 |
| POST | `/api/v1/censers/:id/slosh-analysis` | 执行抗晃荡分析 |
| GET | `/api/v1/censers/:id/slosh-analysis` | 获取抗晃荡分析历史 |
| GET | `/api/v1/censers/:id/frequency-response` | 获取频率响应曲线数据 |
| POST | `/api/v1/censers/:id/gimbal-simulation` | 执行万向环数值仿真 |

### WebSocket

连接地址：`ws://localhost:8080/ws`

推送消息格式：
```json
{
  "type": "sensor_data | alert",
  "data": { ... },
  "time": "2026-06-22T04:42:00Z"
}
```

---

## 告警规则

| 告警类型 | 触发条件 | 默认阈值 | 级别 |
|---------|---------|---------|------|
| `tilt_exceeded` | 炉体倾角 > 阈值 | 15° | warning / critical (>22.5°) |
| `balance_failure` | 平衡评分 < 阈值 | 0.3 | warning / critical (<0.15) |
| `spill_risk` | 洒香概率 > 阈值 | 0.5 | warning / critical (>0.65) |

同类告警 30 秒冷却期，避免刷屏。

---

## 前端功能

- 🎯 **3D 香炉模型**：Three.js 渲染，透明线框展示内外万向环
- 🖱️ **自由视角**：鼠标拖拽旋转，滚轮缩放
- 📊 **实时指标**：平衡评分、洒香风险、炉体倾角动态进度条
- 📈 **三组趋势图**：内外环角度、倾角历史、平衡/风险曲线
- 🔬 **抗晃荡分析**：一键分析步行/骑马/奔跑/乘车/抬轿5种工况
- ⚠️ **告警推送**：实时显示 WebSocket 告警，按严重程度配色
- 🎨 **古金色调UI**：贴合被中香炉金银器文物风格

---

## 参考资料

- 陕西历史博物馆. 唐代葡萄花鸟纹银熏球 [何家村窖藏出土]
- 法门寺博物馆. 鎏金鸿雁纹银熏球 [地宫出土]
- 常平架（Gimbal）刚体动力学理论
- 机械振动：单自由度系统频率响应分析
