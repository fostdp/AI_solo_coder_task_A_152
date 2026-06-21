package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type MechanicalConfig struct {
	Description   string               `json:"description"`
	Reference     string               `json:"reference"`
	Mechanical    MechanicalParams     `json:"mechanical"`
	AlarmThresholds AlarmThresholds     `json:"alarm_thresholds"`
	Presets       []CenserPreset       `json:"presets"`
}

type MechanicalParams struct {
	InnerRing   RingParams       `json:"inner_ring"`
	OuterRing   RingParams       `json:"outer_ring"`
	Body        BodyParams       `json:"body"`
	Bearings    BearingParams    `json:"bearings"`
	Environment EnvironmentParams `json:"environment"`
}

type RingParams struct {
	MassKg          float64 `json:"mass_kg"`
	RadiusM         float64 `json:"radius_m"`
	RotationLimitDeg float64 `json:"rotation_limit_deg"`
	Material        string  `json:"material"`
}

type BodyParams struct {
	MassKg             float64            `json:"mass_kg"`
	RadiusM            float64            `json:"radius_m"`
	MomentsOfInertia  MomentsOfInertia  `json:"moments_of_inertia"`
	Material           string             `json:"material"`
}

type MomentsOfInertia struct {
	Ixx float64 `json:"I_xx"`
	Iyy float64 `json:"I_yy"`
	Izz float64 `json:"I_zz"`
}

type BearingParams struct {
	FrictionCoefficient float64 `json:"friction_coefficient"`
	DampingCoefficient  float64 `json:"damping_coefficient"`
	Type                string  `json:"type"`
}

type EnvironmentParams struct {
	GravityMps2      float64 `json:"gravity_mps2"`
	AirViscosityPas  float64 `json:"air_viscosity_pas"`
	TemperatureC     float64 `json:"temperature_c"`
}

type AlarmThresholds struct {
	TiltAlarmDeg     float64 `json:"tilt_alarm_deg"`
	TiltCriticalDeg  float64 `json:"tilt_critical_deg"`
	BalanceAlarm     float64 `json:"balance_alarm"`
	BalanceCritical  float64 `json:"balance_critical"`
	SpillAlarm       float64 `json:"spill_alarm"`
	SpillCritical    float64 `json:"spill_critical"`
}

type CenserPreset struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ScaleFactor float64 `json:"scale_factor"`
}

type FluidConfig struct {
	Description    string           `json:"description"`
	Reference      string           `json:"reference"`
	Formulas       []PerfumeFormula `json:"perfume_formulas"`
	SloshDynamics  SloshDynamics    `json:"slosh_dynamics"`
	MotionProfiles map[string]MotionProfile `json:"motion_profiles"`
	DefaultFormula string           `json:"default_formula"`
}

type PerfumeFormula struct {
	FormulaID                 string            `json:"formula_id"`
	Name                      string            `json:"name"`
	Description               string            `json:"description"`
	Ingredients               []Ingredient      `json:"ingredients"`
	BaseViscosityPas          float64           `json:"base_viscosity_pas"`
	ViscosityTemperatureCoeff float64           `json:"viscosity_temperature_coeff"`
	DensityKgm3              float64           `json:"density_kgm3"`
	SurfaceTensionNm         float64           `json:"surface_tension_nm"`
}

type Ingredient struct {
	Material      string  `json:"material"`
	Fraction      float64 `json:"fraction"`
	ViscosityPas  float64 `json:"viscosity_pas"`
}

type SloshDynamics struct {
	DefaultFillRatio         float64                  `json:"default_fill_ratio"`
	OptimalFillRatio         float64                  `json:"optimal_fill_ratio"`
	CriticalAngularVelocityRps float64                `json:"critical_angular_velocity_rps"`
	StokesDampingCoeff       float64                  `json:"stokes_damping_coeff"`
	ViscosityDampingExponent float64                  `json:"viscosity_damping_exponent"`
	FillRatioCoefficients    FillRatioCoefficients    `json:"fill_ratio_coefficients"`
}

type FillRatioCoefficients struct {
	Linear    float64 `json:"linear"`
	Quadratic float64 `json:"quadratic"`
}

type MotionProfile struct {
	Name               string  `json:"name"`
	FrequencyHz        float64 `json:"frequency_hz"`
	AmplitudeMps2      float64 `json:"amplitude_mps2"`
	DurationSec        float64 `json:"duration_sec"`
	TypicalUsage       string  `json:"typical_usage"`
	HistoricalReference string `json:"historical_reference"`
}

var (
	Mechanical *MechanicalConfig
	Fluid      *FluidConfig
)

func LoadMechanicalConfig(path string) (*MechanicalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mechanical config: %w", err)
	}

	var cfg MechanicalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse mechanical config: %w", err)
	}

	Mechanical = &cfg
	return &cfg, nil
}

func LoadFluidConfig(path string) (*FluidConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fluid config: %w", err)
	}

	var cfg FluidConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse fluid config: %w", err)
	}

	Fluid = &cfg
	return &cfg, nil
}

func (f *FluidConfig) GetFormula(formulaID string) *PerfumeFormula {
	for i := range f.Formulas {
		if f.Formulas[i].FormulaID == formulaID {
			return &f.Formulas[i]
		}
	}
	return nil
}

func (f *FluidConfig) GetMotionProfile(profileType string) *MotionProfile {
	if profile, ok := f.MotionProfiles[profileType]; ok {
		return &profile
	}
	return nil
}

func (m *MechanicalConfig) GetPreset(code string) *CenserPreset {
	for i := range m.Presets {
		if m.Presets[i].Code == code {
			return &m.Presets[i]
		}
	}
	return nil
}
