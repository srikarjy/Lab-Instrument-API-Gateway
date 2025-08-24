package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// QualityCode represents the quality of a measurement
type QualityCode string

const (
	QualityUnknown     QualityCode = "unknown"
	QualityGood        QualityCode = "good"
	QualityBad         QualityCode = "bad"
	QualityUncertain   QualityCode = "uncertain"
	QualitySubstituted QualityCode = "substituted"
)

// Measurement represents a single measurement data point
type Measurement struct {
	ID             string                 `json:"id" db:"id"`
	DeviceID       string                 `json:"device_id" db:"device_id"`
	Timestamp      time.Time              `json:"timestamp" db:"timestamp"`
	Type           string                 `json:"type" db:"type"`
	Value          float64                `json:"value" db:"value"`
	Unit           string                 `json:"unit" db:"unit"`
	Quality        QualityCode            `json:"quality" db:"quality"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	BatchID        *string                `json:"batch_id" db:"batch_id"`
	SequenceNumber *int                   `json:"sequence_number" db:"sequence_number"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// MeasurementBatch represents a batch of measurements for efficient processing
type MeasurementBatch struct {
	DeviceID     string        `json:"device_id"`
	BatchID      string        `json:"batch_id"`
	Timestamp    time.Time     `json:"timestamp"`
	Measurements []Measurement `json:"measurements"`
	TotalCount   int           `json:"total_count"`
}

// MeasurementStats represents statistical information about measurements
type MeasurementStats struct {
	DeviceID      string    `json:"device_id"`
	Type          string    `json:"type"`
	Count         int64     `json:"count"`
	MinValue      float64   `json:"min_value"`
	MaxValue      float64   `json:"max_value"`
	AvgValue      float64   `json:"avg_value"`
	StdDev        float64   `json:"std_dev"`
	EarliestTime  time.Time `json:"earliest_time"`
	LatestTime    time.Time `json:"latest_time"`
	GoodQuality   int64     `json:"good_quality_count"`
	BadQuality    int64     `json:"bad_quality_count"`
	TotalQuality  int64     `json:"total_quality_count"`
}

// Validate validates the measurement data
func (m *Measurement) Validate() error {
	if m.DeviceID == "" {
		return fmt.Errorf("device ID is required")
	}
	
	if m.Type == "" {
		return fmt.Errorf("measurement type is required")
	}
	
	if m.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	
	// Validate quality code
	validQualities := map[QualityCode]bool{
		QualityUnknown:     true,
		QualityGood:        true,
		QualityBad:         true,
		QualityUncertain:   true,
		QualitySubstituted: true,
	}
	
	if !validQualities[m.Quality] {
		return fmt.Errorf("invalid quality code: %s", m.Quality)
	}
	
	return nil
}

// IsGoodQuality returns true if the measurement has good quality
func (m *Measurement) IsGoodQuality() bool {
	return m.Quality == QualityGood
}

// Age returns the age of the measurement
func (m *Measurement) Age() time.Duration {
	return time.Since(m.Timestamp)
}

// SetDefaults sets default values for the measurement
func (m *Measurement) SetDefaults() {
	if m.Quality == "" {
		m.Quality = QualityUnknown
	}
	
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
}

// Value implements the driver.Valuer interface for QualityCode
func (qc QualityCode) Value() (driver.Value, error) {
	return string(qc), nil
}

// Scan implements the sql.Scanner interface for QualityCode
func (qc *QualityCode) Scan(value interface{}) error {
	if value == nil {
		*qc = QualityUnknown
		return nil
	}
	
	switch v := value.(type) {
	case string:
		*qc = QualityCode(v)
	case []byte:
		*qc = QualityCode(v)
	default:
		return fmt.Errorf("cannot scan %T into QualityCode", value)
	}
	
	return nil
}

// MarshalJSON implements json.Marshaler for QualityCode
func (qc QualityCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(qc))
}

// UnmarshalJSON implements json.Unmarshaler for QualityCode
func (qc *QualityCode) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*qc = QualityCode(s)
	return nil
}

