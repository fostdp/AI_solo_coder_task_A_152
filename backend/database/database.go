package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"censer-simulation/models"
)

var globalDB *DB

type DB struct {
	pool *pgxpool.Pool
}

func InitDB() error {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/censer_sim?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	globalDB = &DB{pool: pool}
	return nil
}

func GetDB() *DB {
	return globalDB
}

func CloseDB() {
	if globalDB != nil && globalDB.pool != nil {
		globalDB.pool.Close()
	}
}

func (db *DB) InsertSensorData(data *models.SensorData) error {
	query := `
		INSERT INTO sensor_data (
			time, censer_id, inner_ring_angle, outer_ring_angle, body_tilt,
			slosh_acceleration, inner_ring_velocity, outer_ring_velocity,
			body_angular_velocity, temperature, balance_score, spill_risk, raw_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := db.pool.Exec(context.Background(), query,
		data.Time, data.CenserID, data.InnerRingAngle, data.OuterRingAngle,
		data.BodyTilt, data.SloshAcceleration, data.InnerRingVelocity,
		data.OuterRingVelocity, data.BodyAngularVelocity, data.Temperature,
		data.BalanceScore, data.SpillRisk, data.RawData,
	)
	return err
}

func (db *DB) InsertAlert(ctx context.Context, alert *models.Alert) error {
	query := `
		INSERT INTO alerts (
			censer_id, alert_type, severity, message, threshold_value,
			actual_value, sensor_data_time, acknowledged, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	return db.pool.QueryRow(ctx, query,
		alert.CenserID, alert.AlertType, alert.Severity, alert.Message,
		alert.ThresholdValue, alert.ActualValue, alert.SensorDataTime,
		false, time.Now(),
	).Scan(&alert.ID)
}

func (db *DB) GetCensers(ctx context.Context) ([]models.Censer, error) {
	query := `SELECT id, name, code, description, created_at, updated_at FROM censers ORDER BY code`
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var censers []models.Censer
	for rows.Next() {
		var c models.Censer
		err := rows.Scan(&c.ID, &c.Name, &c.Code, &c.Description, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		censers = append(censers, c)
	}
	return censers, rows.Err()
}

func (db *DB) GetCenserByCode(code string) (*models.Censer, error) {
	query := `SELECT id, name, code, description, created_at, updated_at FROM censers WHERE code = $1`
	var c models.Censer
	err := db.pool.QueryRow(context.Background(), query, code).Scan(
		&c.ID, &c.Name, &c.Code, &c.Description, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) GetSimulationConfig(censerID uuid.UUID) (*models.SimulationConfig, error) {
	query := `
		SELECT id, censer_id, inner_ring_mass, outer_ring_mass, body_mass,
			inner_ring_radius, outer_ring_radius, body_radius, friction_coefficient,
			damping_coefficient, gravity, tilt_alarm_threshold, balance_alarm_threshold,
			spill_alarm_threshold, perfume_viscosity, fill_ratio, created_at, updated_at
		FROM simulation_configs WHERE censer_id = $1
	`
	var cfg models.SimulationConfig
	err := db.pool.QueryRow(context.Background(), query, censerID).Scan(
		&cfg.ID, &cfg.CenserID, &cfg.InnerRingMass, &cfg.OuterRingMass, &cfg.BodyMass,
		&cfg.InnerRingRadius, &cfg.OuterRingRadius, &cfg.BodyRadius, &cfg.FrictionCoefficient,
		&cfg.DampingCoefficient, &cfg.Gravity, &cfg.TiltAlarmThreshold, &cfg.BalanceAlarmThreshold,
		&cfg.SpillAlarmThreshold, &cfg.PerfumeViscosity, &cfg.FillRatio, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (db *DB) GetLatestSensorData(ctx context.Context) ([]models.LatestSensorData, error) {
	query := `
		SELECT time, censer_id, censer_name, censer_code, inner_ring_angle,
			outer_ring_angle, body_tilt, slosh_acceleration, balance_score, spill_risk
		FROM v_latest_sensor_data ORDER BY censer_code
	`
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.LatestSensorData
	for rows.Next() {
		var r models.LatestSensorData
		err := rows.Scan(&r.Time, &r.CenserID, &r.CenserName, &r.CenserCode,
			&r.InnerRingAngle, &r.OuterRingAngle, &r.BodyTilt, &r.SloshAcceleration,
			&r.BalanceScore, &r.SpillRisk)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetSensorDataByCenser(ctx context.Context, censerID uuid.UUID, limit int) ([]models.SensorData, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT time, censer_id, inner_ring_angle, outer_ring_angle, body_tilt,
			slosh_acceleration, inner_ring_velocity, outer_ring_velocity,
			body_angular_velocity, temperature, balance_score, spill_risk
		FROM sensor_data WHERE censer_id = $1 ORDER BY time DESC LIMIT $2
	`
	rows, err := db.pool.Query(ctx, query, censerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SensorData
	for rows.Next() {
		var r models.SensorData
		err := rows.Scan(&r.Time, &r.CenserID, &r.InnerRingAngle, &r.OuterRingAngle,
			&r.BodyTilt, &r.SloshAcceleration, &r.InnerRingVelocity, &r.OuterRingVelocity,
			&r.BodyAngularVelocity, &r.Temperature, &r.BalanceScore, &r.SpillRisk)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetStabilityStats(ctx context.Context) ([]models.StabilityStats, error) {
	query := `
		SELECT censer_id, censer_name, censer_code, data_points, avg_tilt,
			max_tilt, min_balance_score, avg_balance_score, avg_spill_risk, max_spill_risk
		FROM v_stability_stats ORDER BY censer_code
	`
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.StabilityStats
	for rows.Next() {
		var r models.StabilityStats
		err := rows.Scan(&r.CenserID, &r.CenserName, &r.CenserCode, &r.DataPoints,
			&r.AvgTilt, &r.MaxTilt, &r.MinBalanceScore, &r.AvgBalanceScore,
			&r.AvgSpillRisk, &r.MaxSpillRisk)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetActiveAlerts(ctx context.Context) ([]models.ActiveAlerts, error) {
	query := `
		SELECT censer_id, censer_name, censer_code, critical_count, warning_count, total_count
		FROM v_active_alerts ORDER BY censer_code
	`
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.ActiveAlerts
	for rows.Next() {
		var r models.ActiveAlerts
		err := rows.Scan(&r.CenserID, &r.CenserName, &r.CenserCode,
			&r.CriticalCount, &r.WarningCount, &r.TotalCount)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) GetAlertsByCenser(ctx context.Context, censerID uuid.UUID, limit int) ([]models.Alert, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, censer_id, alert_type, severity, message, threshold_value,
			actual_value, sensor_data_time, acknowledged, acknowledged_at,
			acknowledged_by, created_at
		FROM alerts WHERE censer_id = $1 ORDER BY created_at DESC LIMIT $2
	`
	rows, err := db.pool.Query(ctx, query, censerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.Alert
	for rows.Next() {
		var r models.Alert
		err := rows.Scan(&r.ID, &r.CenserID, &r.AlertType, &r.Severity, &r.Message,
			&r.ThresholdValue, &r.ActualValue, &r.SensorDataTime, &r.Acknowledged,
			&r.AcknowledgedAt, &r.AcknowledgedBy, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) InsertSloshAnalysis(ctx context.Context, analysis *models.SloshAnalysis) error {
	query := `
		INSERT INTO slosh_analysis (
			censer_id, analysis_type, motion_type, frequency, amplitude,
			damping_ratio, resonance_factor, max_tilt_angle, spill_probability,
			balance_efficiency, analysis_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`
	return db.pool.QueryRow(ctx, query,
		analysis.CenserID, analysis.AnalysisType, analysis.MotionType,
		analysis.Frequency, analysis.Amplitude, analysis.DampingRatio,
		analysis.ResonanceFactor, analysis.MaxTiltAngle, analysis.SpillProbability,
		analysis.BalanceEfficiency, analysis.AnalysisData,
	).Scan(&analysis.ID, &analysis.CreatedAt)
}

func (db *DB) GetSloshAnalysisByCenser(ctx context.Context, censerID uuid.UUID, limit int) ([]models.SloshAnalysis, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT id, censer_id, analysis_type, motion_type, frequency, amplitude,
			damping_ratio, resonance_factor, max_tilt_angle, spill_probability,
			balance_efficiency, analysis_data, created_at
		FROM slosh_analysis WHERE censer_id = $1 ORDER BY created_at DESC LIMIT $2
	`
	rows, err := db.pool.Query(ctx, query, censerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SloshAnalysis
	for rows.Next() {
		var r models.SloshAnalysis
		err := rows.Scan(&r.ID, &r.CenserID, &r.AnalysisType, &r.MotionType,
			&r.Frequency, &r.Amplitude, &r.DampingRatio, &r.ResonanceFactor,
			&r.MaxTiltAngle, &r.SpillProbability, &r.BalanceEfficiency,
			&r.AnalysisData, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) AcknowledgeAlert(ctx context.Context, alertID uuid.UUID, acknowledgedBy string) error {
	query := `
		UPDATE alerts SET acknowledged = TRUE, acknowledged_at = $1, acknowledged_by = $2
		WHERE id = $3
	`
	_, err := db.pool.Exec(ctx, query, time.Now(), acknowledgedBy, alertID)
	return err
}

func InsertSensorData(ctx context.Context, data *models.SensorData) error {
	if globalDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return globalDB.InsertSensorData(data)
}

func InsertAlert(ctx context.Context, alert *models.Alert) error {
	if globalDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return globalDB.InsertAlert(ctx, alert)
}

func GetCensers(ctx context.Context) ([]models.Censer, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetCensers(ctx)
}

func GetCenserByCode(ctx context.Context, code string) (*models.Censer, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetCenserByCode(code)
}

func GetSimulationConfig(ctx context.Context, censerID uuid.UUID) (*models.SimulationConfig, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetSimulationConfig(censerID)
}

func GetLatestSensorData(ctx context.Context) ([]models.LatestSensorData, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetLatestSensorData(ctx)
}

func GetSensorDataByCenser(ctx context.Context, censerID uuid.UUID, limit int) ([]models.SensorData, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetSensorDataByCenser(ctx, censerID, limit)
}

func GetStabilityStats(ctx context.Context) ([]models.StabilityStats, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetStabilityStats(ctx)
}

func GetActiveAlerts(ctx context.Context) ([]models.ActiveAlerts, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetActiveAlerts(ctx)
}

func GetAlertsByCenser(ctx context.Context, censerID uuid.UUID, limit int) ([]models.Alert, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetAlertsByCenser(ctx, censerID, limit)
}

func InsertSloshAnalysis(ctx context.Context, analysis *models.SloshAnalysis) error {
	if globalDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return globalDB.InsertSloshAnalysis(ctx, analysis)
}

func GetSloshAnalysisByCenser(ctx context.Context, censerID uuid.UUID, limit int) ([]models.SloshAnalysis, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return globalDB.GetSloshAnalysisByCenser(ctx, censerID, limit)
}

func AcknowledgeAlert(ctx context.Context, alertID uuid.UUID, acknowledgedBy string) error {
	if globalDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return globalDB.AcknowledgeAlert(ctx, alertID, acknowledgedBy)
}
