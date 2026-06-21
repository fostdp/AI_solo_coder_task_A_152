-- ============================================================
-- 古代被中香炉（银熏球）万向平衡机构仿真与抗晃荡分析系统
-- TimescaleDB 初始化脚本
-- ============================================================

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- 香炉设备表
-- ============================================================
CREATE TABLE IF NOT EXISTS censers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 传感器数据表（核心时序表）
-- ============================================================
CREATE TABLE IF NOT EXISTS sensor_data (
    time TIMESTAMPTZ NOT NULL,
    censer_id UUID NOT NULL REFERENCES censers(id) ON DELETE CASCADE,
    inner_ring_angle DOUBLE PRECISION NOT NULL,
    outer_ring_angle DOUBLE PRECISION NOT NULL,
    body_tilt DOUBLE PRECISION NOT NULL,
    slosh_acceleration DOUBLE PRECISION NOT NULL,
    inner_ring_velocity DOUBLE PRECISION,
    outer_ring_velocity DOUBLE PRECISION,
    body_angular_velocity DOUBLE PRECISION,
    temperature DOUBLE PRECISION,
    balance_score DOUBLE PRECISION,
    spill_risk DOUBLE PRECISION,
    raw_data JSONB
);

-- 创建hypertable（TimescaleDB核心特性）
SELECT create_hypertable('sensor_data', 'time', if_not_exists => TRUE);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_sensor_data_censer_id ON sensor_data (censer_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_data_balance_score ON sensor_data (censer_id, time DESC) WHERE balance_score < 0.5;
CREATE INDEX IF NOT EXISTS idx_sensor_data_spill_risk ON sensor_data (censer_id, time DESC) WHERE spill_risk > 0.3;

-- ============================================================
-- 告警记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    censer_id UUID NOT NULL REFERENCES censers(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    message TEXT NOT NULL,
    threshold_value DOUBLE PRECISION,
    actual_value DOUBLE PRECISION,
    sensor_data_time TIMESTAMPTZ,
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alerts_censer_id ON alerts (censer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts (severity, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_unacknowledged ON alerts (acknowledged) WHERE acknowledged = FALSE;

-- ============================================================
-- 仿真配置参数表
-- ============================================================
CREATE TABLE IF NOT EXISTS simulation_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    censer_id UUID UNIQUE REFERENCES censers(id) ON DELETE CASCADE,
    inner_ring_mass DOUBLE PRECISION NOT NULL DEFAULT 0.05,
    outer_ring_mass DOUBLE PRECISION NOT NULL DEFAULT 0.08,
    body_mass DOUBLE PRECISION NOT NULL DEFAULT 0.15,
    inner_ring_radius DOUBLE PRECISION NOT NULL DEFAULT 0.04,
    outer_ring_radius DOUBLE PRECISION NOT NULL DEFAULT 0.05,
    body_radius DOUBLE PRECISION NOT NULL DEFAULT 0.03,
    friction_coefficient DOUBLE PRECISION NOT NULL DEFAULT 0.05,
    damping_coefficient DOUBLE PRECISION NOT NULL DEFAULT 0.15,
    gravity DOUBLE PRECISION NOT NULL DEFAULT 9.81,
    tilt_alarm_threshold DOUBLE PRECISION NOT NULL DEFAULT 15.0,
    balance_alarm_threshold DOUBLE PRECISION NOT NULL DEFAULT 0.3,
    spill_alarm_threshold DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    perfume_viscosity DOUBLE PRECISION NOT NULL DEFAULT 0.5,
    fill_ratio DOUBLE PRECISION NOT NULL DEFAULT 0.6,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='simulation_configs' AND column_name='perfume_viscosity') THEN
        ALTER TABLE simulation_configs ADD COLUMN perfume_viscosity DOUBLE PRECISION NOT NULL DEFAULT 0.5;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='simulation_configs' AND column_name='fill_ratio') THEN
        ALTER TABLE simulation_configs ADD COLUMN fill_ratio DOUBLE PRECISION NOT NULL DEFAULT 0.6;
    END IF;
END $$;

-- ============================================================
-- 抗晃荡分析结果表
-- ============================================================
CREATE TABLE IF NOT EXISTS slosh_analysis (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    censer_id UUID NOT NULL REFERENCES censers(id) ON DELETE CASCADE,
    analysis_type VARCHAR(50) NOT NULL,
    motion_type VARCHAR(50) NOT NULL,
    frequency DOUBLE PRECISION NOT NULL,
    amplitude DOUBLE PRECISION NOT NULL,
    damping_ratio DOUBLE PRECISION,
    resonance_factor DOUBLE PRECISION,
    max_tilt_angle DOUBLE PRECISION,
    spill_probability DOUBLE PRECISION,
    balance_efficiency DOUBLE PRECISION,
    analysis_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_slosh_analysis_censer_id ON slosh_analysis (censer_id, created_at DESC);

-- ============================================================
-- 视图：最近传感器数据
-- ============================================================
CREATE OR REPLACE VIEW v_latest_sensor_data AS
SELECT DISTINCT ON (censer_id)
    sd.time,
    sd.censer_id,
    c.name AS censer_name,
    c.code AS censer_code,
    sd.inner_ring_angle,
    sd.outer_ring_angle,
    sd.body_tilt,
    sd.slosh_acceleration,
    sd.balance_score,
    sd.spill_risk
FROM sensor_data sd
JOIN censers c ON c.id = sd.censer_id
ORDER BY censer_id, time DESC;

-- ============================================================
-- 视图：活动告警统计
-- ============================================================
CREATE OR REPLACE VIEW v_active_alerts AS
SELECT
    c.id AS censer_id,
    c.name AS censer_name,
    c.code AS censer_code,
    COUNT(a.id) FILTER (WHERE a.severity = 'critical') AS critical_count,
    COUNT(a.id) FILTER (WHERE a.severity = 'warning') AS warning_count,
    COUNT(a.id) AS total_count
FROM censers c
LEFT JOIN alerts a ON a.censer_id = c.id AND a.acknowledged = FALSE
GROUP BY c.id, c.name, c.code;

-- ============================================================
-- 视图：香炉稳定性统计（1小时窗口）
-- ============================================================
CREATE OR REPLACE VIEW v_stability_stats AS
SELECT
    c.id AS censer_id,
    c.name AS censer_name,
    c.code AS censer_code,
    COUNT(sd.id) AS data_points,
    AVG(sd.body_tilt) AS avg_tilt,
    MAX(sd.body_tilt) AS max_tilt,
    MIN(sd.balance_score) AS min_balance_score,
    AVG(sd.balance_score) AS avg_balance_score,
    AVG(sd.spill_risk) AS avg_spill_risk,
    MAX(sd.spill_risk) AS max_spill_risk
FROM censers c
LEFT JOIN sensor_data sd ON sd.censer_id = c.id
    AND sd.time > NOW() - INTERVAL '1 hour'
GROUP BY c.id, c.name, c.code;

-- ============================================================
-- 连续聚合：5分钟平衡指标聚合
-- ============================================================
SELECT create_hypertable(
    'sensor_data',
    'time',
    if_not_exists => TRUE
);

CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_data_5m
WITH (timescaledb.continuous) AS
SELECT
    censer_id,
    time_bucket('5 minutes', time) AS bucket,
    AVG(inner_ring_angle) AS avg_inner_ring_angle,
    MAX(ABS(inner_ring_angle)) AS max_inner_ring_angle,
    AVG(outer_ring_angle) AS avg_outer_ring_angle,
    MAX(ABS(outer_ring_angle)) AS max_outer_ring_angle,
    AVG(body_tilt) AS avg_body_tilt,
    MAX(ABS(body_tilt)) AS max_body_tilt,
    AVG(slosh_acceleration) AS avg_slosh_acceleration,
    MAX(slosh_acceleration) AS max_slosh_acceleration,
    AVG(balance_score) AS avg_balance_score,
    MIN(balance_score) AS min_balance_score,
    AVG(spill_risk) AS avg_spill_risk,
    MAX(spill_risk) AS max_spill_risk,
    COUNT(*) AS sample_count
FROM sensor_data
GROUP BY censer_id, time_bucket('5 minutes', time)
WITH NO DATA;

-- 启用实时聚合
ALTER MATERIALIZED VIEW sensor_data_5m SET (timescaledb.materialized_only = false);

-- ============================================================
-- 初始数据：插入示例香炉
-- ============================================================
INSERT INTO censers (name, code, description) VALUES
    ('唐代葡萄花鸟纹银熏球', 'CENSER-001', '陕西历史博物馆藏，直径约4.6cm，1970年西安何家村出土'),
    ('唐代鎏金鸿雁纹银熏球', 'CENSER-002', '法门寺地宫出土，鎏金工艺，饰鸿雁纹'),
    ('复原模型A型', 'CENSER-003', '工艺史团队复原研究模型，标准万向平衡结构')
ON CONFLICT (code) DO NOTHING;

-- 为每个香炉创建默认仿真配置
INSERT INTO simulation_configs (censer_id)
SELECT id FROM censers
WHERE id NOT IN (SELECT censer_id FROM simulation_configs);

-- ============================================================
-- 自动更新时间戳触发器
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS censers_update_updated_at ON censers;
CREATE TRIGGER censers_update_updated_at
    BEFORE UPDATE ON censers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS configs_update_updated_at ON simulation_configs;
CREATE TRIGGER configs_update_updated_at
    BEFORE UPDATE ON simulation_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ============================================================
-- 连续聚合：1小时聚合
-- ============================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_data_1h
WITH (timescaledb.continuous) AS
SELECT
    censer_id,
    time_bucket('1 hour', time) AS bucket,
    AVG(inner_ring_angle) AS avg_inner_ring_angle,
    MAX(ABS(inner_ring_angle)) AS max_inner_ring_angle,
    AVG(outer_ring_angle) AS avg_outer_ring_angle,
    MAX(ABS(outer_ring_angle)) AS max_outer_ring_angle,
    AVG(body_tilt) AS avg_body_tilt,
    MAX(ABS(body_tilt)) AS max_body_tilt,
    AVG(slosh_acceleration) AS avg_slosh_acceleration,
    MAX(slosh_acceleration) AS max_slosh_acceleration,
    AVG(balance_score) AS avg_balance_score,
    MIN(balance_score) AS min_balance_score,
    AVG(spill_risk) AS avg_spill_risk,
    MAX(spill_risk) AS max_spill_risk,
    COUNT(*) AS sample_count
FROM sensor_data
GROUP BY censer_id, time_bucket('1 hour', time)
WITH NO DATA;

ALTER MATERIALIZED VIEW sensor_data_1h SET (timescaledb.materialized_only = false);

-- ============================================================
-- 连续聚合：1天聚合
-- ============================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_data_1d
WITH (timescaledb.continuous) AS
SELECT
    censer_id,
    time_bucket('1 day', time) AS bucket,
    AVG(inner_ring_angle) AS avg_inner_ring_angle,
    MAX(ABS(inner_ring_angle)) AS max_inner_ring_angle,
    AVG(outer_ring_angle) AS avg_outer_ring_angle,
    MAX(ABS(outer_ring_angle)) AS max_outer_ring_angle,
    AVG(body_tilt) AS avg_body_tilt,
    MAX(ABS(body_tilt)) AS max_body_tilt,
    AVG(slosh_acceleration) AS avg_slosh_acceleration,
    MAX(slosh_acceleration) AS max_slosh_acceleration,
    AVG(balance_score) AS avg_balance_score,
    MIN(balance_score) AS min_balance_score,
    AVG(spill_risk) AS avg_spill_risk,
    MAX(spill_risk) AS max_spill_risk,
    COUNT(*) AS sample_count
FROM sensor_data
GROUP BY censer_id, time_bucket('1 day', time)
WITH NO DATA;

ALTER MATERIALIZED VIEW sensor_data_1d SET (timescaledb.materialized_only = false);

-- ============================================================
-- 数据保留策略
-- ============================================================
-- 原始数据：保留7天
SELECT add_retention_policy('sensor_data', INTERVAL '7 days', if_not_exists => TRUE);

-- 5分钟聚合：保留30天
SELECT add_retention_policy('sensor_data_5m', INTERVAL '30 days', if_not_exists => TRUE);

-- 1小时聚合：保留1年
SELECT add_retention_policy('sensor_data_1h', INTERVAL '1 year', if_not_exists => TRUE);

-- 1天聚合：永久保留（不设置保留策略）

-- ============================================================
-- 连续聚合自动刷新策略
-- ============================================================
-- 5分钟聚合：每5分钟刷新一次，刷新最近1小时数据
SELECT add_continuous_aggregate_policy(
    'sensor_data_5m',
    start_offset => INTERVAL '1 hour',
    end_offset => INTERVAL '5 minutes',
    schedule_interval => INTERVAL '5 minutes',
    if_not_exists => TRUE
);

-- 1小时聚合：每30分钟刷新一次，刷新最近6小时数据
SELECT add_continuous_aggregate_policy(
    'sensor_data_1h',
    start_offset => INTERVAL '6 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '30 minutes',
    if_not_exists => TRUE
);

-- 1天聚合：每6小时刷新一次，刷新最近2天数据
SELECT add_continuous_aggregate_policy(
    'sensor_data_1d',
    start_offset => INTERVAL '2 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '6 hours',
    if_not_exists => TRUE
);

-- ============================================================
-- 告警数据保留策略
-- ============================================================
-- 注意：alerts 表非hypertable，保留策略需通过pg_cron定期清理
-- 建议保留90天告警记录，以下SQL可配合pg_cron使用：
-- SELECT cron.schedule('clean-old-alerts', '0 2 * * *', 
--   $$DELETE FROM alerts WHERE created_at < NOW() - INTERVAL ''90 days'';$$);
-- ============================================================

-- ============================================================
-- 压缩策略（可选，节省存储空间）
-- ============================================================
-- 原始数据：3天后压缩
-- 注意：压缩后数据为只读，会影响某些查询性能
-- SELECT add_compression_policy('sensor_data', INTERVAL '3 days', if_not_exists => TRUE);

-- ============================================================
-- 数据降采样层级总结
-- ============================================================
-- | 聚合粒度 | 保留时长  | 刷新间隔  |
-- |----------|-----------|-----------|
-- | raw      | 7天       | -         |
-- | 5分钟    | 30天      | 5分钟     |
-- | 1小时    | 1年       | 30分钟    |
-- | 1天      | 永久      | 6小时     |
-- ============================================================
