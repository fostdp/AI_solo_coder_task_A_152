package models

import (
	"time"

	"github.com/google/uuid"
)

type Censer struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Code        string    `json:"code" db:"code"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type SensorData struct {
	Time                time.Time  `json:"time" db:"time"`
	CenserID            uuid.UUID  `json:"censer_id" db:"censer_id"`
	InnerRingAngle      float64    `json:"inner_ring_angle" db:"inner_ring_angle"`
	OuterRingAngle      float64    `json:"outer_ring_angle" db:"outer_ring_angle"`
	BodyTilt            float64    `json:"body_tilt" db:"body_tilt"`
	SloshAcceleration   float64    `json:"slosh_acceleration" db:"slosh_acceleration"`
	InnerRingVelocity   *float64   `json:"inner_ring_velocity,omitempty" db:"inner_ring_velocity"`
	OuterRingVelocity   *float64   `json:"outer_ring_velocity,omitempty" db:"outer_ring_velocity"`
	BodyAngularVelocity *float64   `json:"body_angular_velocity,omitempty" db:"body_angular_velocity"`
	Temperature         *float64   `json:"temperature,omitempty" db:"temperature"`
	BalanceScore        *float64   `json:"balance_score,omitempty" db:"balance_score"`
	SpillRisk           *float64   `json:"spill_risk,omitempty" db:"spill_risk"`
	RawData             *string    `json:"raw_data,omitempty" db:"raw_data"`
}

type Alert struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	CenserID       uuid.UUID  `json:"censer_id" db:"censer_id"`
	AlertType      string     `json:"alert_type" db:"alert_type"`
	Severity       string     `json:"severity" db:"severity"`
	Message        string     `json:"message" db:"message"`
	ThresholdValue *float64   `json:"threshold_value,omitempty" db:"threshold_value"`
	ActualValue    *float64   `json:"actual_value,omitempty" db:"actual_value"`
	SensorDataTime *time.Time `json:"sensor_data_time,omitempty" db:"sensor_data_time"`
	Acknowledged   bool       `json:"acknowledged" db:"acknowledged"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	AcknowledgedBy *string    `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

type SimulationConfig struct {
	ID                    uuid.UUID `json:"id" db:"id"`
	CenserID              uuid.UUID `json:"censer_id" db:"censer_id"`
	InnerRingMass         float64   `json:"inner_ring_mass" db:"inner_ring_mass"`
	OuterRingMass         float64   `json:"outer_ring_mass" db:"outer_ring_mass"`
	BodyMass              float64   `json:"body_mass" db:"body_mass"`
	InnerRingRadius       float64   `json:"inner_ring_radius" db:"inner_ring_radius"`
	OuterRingRadius       float64   `json:"outer_ring_radius" db:"outer_ring_radius"`
	BodyRadius            float64   `json:"body_radius" db:"body_radius"`
	FrictionCoefficient   float64   `json:"friction_coefficient" db:"friction_coefficient"`
	DampingCoefficient    float64   `json:"damping_coefficient" db:"damping_coefficient"`
	Gravity               float64   `json:"gravity" db:"gravity"`
	TiltAlarmThreshold    float64   `json:"tilt_alarm_threshold" db:"tilt_alarm_threshold"`
	BalanceAlarmThreshold float64   `json:"balance_alarm_threshold" db:"balance_alarm_threshold"`
	SpillAlarmThreshold   float64   `json:"spill_alarm_threshold" db:"spill_alarm_threshold"`
	PerfumeViscosity      float64   `json:"perfume_viscosity" db:"perfume_viscosity"`
	FillRatio             float64   `json:"fill_ratio" db:"fill_ratio"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

type SloshAnalysis struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	CenserID          uuid.UUID  `json:"censer_id" db:"censer_id"`
	AnalysisType      string     `json:"analysis_type" db:"analysis_type"`
	MotionType        string     `json:"motion_type" db:"motion_type"`
	Frequency         float64    `json:"frequency" db:"frequency"`
	Amplitude         float64    `json:"amplitude" db:"amplitude"`
	DampingRatio      *float64   `json:"damping_ratio,omitempty" db:"damping_ratio"`
	ResonanceFactor   *float64   `json:"resonance_factor,omitempty" db:"resonance_factor"`
	MaxTiltAngle      *float64   `json:"max_tilt_angle,omitempty" db:"max_tilt_angle"`
	SpillProbability  *float64   `json:"spill_probability,omitempty" db:"spill_probability"`
	BalanceEfficiency *float64   `json:"balance_efficiency,omitempty" db:"balance_efficiency"`
	AnalysisData      *string    `json:"analysis_data,omitempty" db:"analysis_data"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

type LatestSensorData struct {
	Time              time.Time `json:"time" db:"time"`
	CenserID          uuid.UUID `json:"censer_id" db:"censer_id"`
	CenserName        string    `json:"censer_name" db:"censer_name"`
	CenserCode        string    `json:"censer_code" db:"censer_code"`
	InnerRingAngle    float64   `json:"inner_ring_angle" db:"inner_ring_angle"`
	OuterRingAngle    float64   `json:"outer_ring_angle" db:"outer_ring_angle"`
	BodyTilt          float64   `json:"body_tilt" db:"body_tilt"`
	SloshAcceleration float64   `json:"slosh_acceleration" db:"slosh_acceleration"`
	BalanceScore      *float64  `json:"balance_score,omitempty" db:"balance_score"`
	SpillRisk         *float64  `json:"spill_risk,omitempty" db:"spill_risk"`
}

type StabilityStats struct {
	CenserID         uuid.UUID `json:"censer_id" db:"censer_id"`
	CenserName       string    `json:"censer_name" db:"censer_name"`
	CenserCode       string    `json:"censer_code" db:"censer_code"`
	DataPoints       int64     `json:"data_points" db:"data_points"`
	AvgTilt          *float64  `json:"avg_tilt,omitempty" db:"avg_tilt"`
	MaxTilt          *float64  `json:"max_tilt,omitempty" db:"max_tilt"`
	MinBalanceScore  *float64  `json:"min_balance_score,omitempty" db:"min_balance_score"`
	AvgBalanceScore  *float64  `json:"avg_balance_score,omitempty" db:"avg_balance_score"`
	AvgSpillRisk     *float64  `json:"avg_spill_risk,omitempty" db:"avg_spill_risk"`
	MaxSpillRisk     *float64  `json:"max_spill_risk,omitempty" db:"max_spill_risk"`
}

type ActiveAlerts struct {
	CenserID      uuid.UUID `json:"censer_id" db:"censer_id"`
	CenserName    string    `json:"censer_name" db:"censer_name"`
	CenserCode    string    `json:"censer_code" db:"censer_code"`
	CriticalCount int64     `json:"critical_count" db:"critical_count"`
	WarningCount  int64     `json:"warning_count" db:"warning_count"`
	TotalCount    int64     `json:"total_count" db:"total_count"`
}

type WebsocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Time    time.Time   `json:"time"`
}

type GimbalState struct {
	InnerAngle     float64
	OuterAngle     float64
	BodyAngle      float64
	InnerVelocity  float64
	OuterVelocity  float64
	BodyVelocity   float64
}

type ExternalForce struct {
	AccelerationX float64
	AccelerationY float64
	AccelerationZ float64
	RotationRate  float64
}

type SloshAnalysisResult struct {
	MotionType        string    `json:"motion_type"`
	Frequency         float64   `json:"frequency"`
	Amplitude         float64   `json:"amplitude"`
	DampingRatio      float64   `json:"damping_ratio"`
	ResonanceFactor   float64   `json:"resonance_factor"`
	MaxTiltAngle      float64   `json:"max_tilt_angle"`
	SpillProbability  float64   `json:"spill_probability"`
	BalanceEfficiency float64   `json:"balance_efficiency"`
	TimeSeries        []float64 `json:"time_series,omitempty"`
}
