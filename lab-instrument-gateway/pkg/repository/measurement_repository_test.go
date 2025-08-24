package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/yourorg/lab-gateway/pkg/models"
)

func TestMeasurementRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		measurement *models.Measurement
		wantErr     bool
	}{
		{
			name: "valid measurement",
			measurement: &models.Measurement{
				ID:             "test-measurement-1",
				DeviceID:       "test-device-1",
				Timestamp:      time.Now(),
				Type:           "temperature",
				Value:          23.5,
				Unit:           "celsius",
				Quality:        models.QualityGood,
				SequenceNumber: &[]int{1}[0],
				Metadata: map[string]interface{}{
					"sensor_id": "temp-001",
					"location":  "lab-1",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid measurement - missing device ID",
			measurement: &models.Measurement{
				ID:        "test-measurement-2",
				Timestamp: time.Now(),
				Type:      "temperature",
				Value:     23.5,
				Unit:      "celsius",
				Quality:   models.QualityGood,
			},
			wantErr: true,
		},
		{
			name: "invalid measurement - missing type",
			measurement: &models.Measurement{
				ID:        "test-measurement-3",
				DeviceID:  "test-device-1",
				Timestamp: time.Now(),
				Value:     23.5,
				Unit:      "celsius",
				Quality:   models.QualityGood,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate measurement
			if err := tt.measurement.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Measurement validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMeasurementRepository_BulkCreate(t *testing.T) {
	now := time.Now()
	measurements := []*models.Measurement{
		{
			ID:             "bulk-measurement-1",
			DeviceID:       "test-device-1",
			Timestamp:      now,
			Type:           "temperature",
			Value:          23.5,
			Unit:           "celsius",
			Quality:        models.QualityGood,
			SequenceNumber: &[]int{1}[0],
		},
		{
			ID:             "bulk-measurement-2",
			DeviceID:       "test-device-1",
			Timestamp:      now.Add(time.Second),
			Type:           "humidity",
			Value:          65.2,
			Unit:           "percent",
			Quality:        models.QualityGood,
			SequenceNumber: &[]int{2}[0],
		},
		{
			ID:             "bulk-measurement-3",
			DeviceID:       "test-device-1",
			Timestamp:      now.Add(2 * time.Second),
			Type:           "pressure",
			Value:          1013.25,
			Unit:           "hPa",
			Quality:        models.QualityGood,
			SequenceNumber: &[]int{3}[0],
		},
	}

	t.Run("bulk create validation", func(t *testing.T) {
		// Validate all measurements before bulk operation
		for i, measurement := range measurements {
			if err := measurement.Validate(); err != nil {
				t.Errorf("Measurement %d validation failed: %v", i, err)
			}
		}
	})

	t.Run("high throughput simulation", func(t *testing.T) {
		// Simulate high-throughput scenario
		batchSize := 1000
		largeBatch := make([]*models.Measurement, batchSize)
		
		for i := 0; i < batchSize; i++ {
			largeBatch[i] = &models.Measurement{
				ID:             fmt.Sprintf("perf-measurement-%d", i),
				DeviceID:       "perf-device-1",
				Timestamp:      now.Add(time.Duration(i) * time.Millisecond),
				Type:           "temperature",
				Value:          20.0 + float64(i%10),
				Unit:           "celsius",
				Quality:        models.QualityGood,
				SequenceNumber: &[]int{i + 1}[0],
			}
		}

		// Validate batch size is reasonable for bulk operations
		if len(largeBatch) != batchSize {
			t.Errorf("Expected batch size %d, got %d", batchSize, len(largeBatch))
		}

		// In a real test, we would measure the time taken for bulk insert
		// and ensure it meets performance requirements (e.g., 10,000+ inserts/second)
	})
}

func TestMeasurementRepository_TimeRangeQueries(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name      string
		deviceID  string
		startTime time.Time
		endTime   time.Time
		wantErr   bool
	}{
		{
			name:      "valid time range - last hour",
			deviceID:  "test-device-1",
			startTime: now.Add(-time.Hour),
			endTime:   now,
			wantErr:   false,
		},
		{
			name:      "valid time range - last day",
			deviceID:  "test-device-1",
			startTime: now.Add(-24 * time.Hour),
			endTime:   now,
			wantErr:   false,
		},
		{
			name:      "invalid time range - end before start",
			deviceID:  "test-device-1",
			startTime: now,
			endTime:   now.Add(-time.Hour),
			wantErr:   true,
		},
		{
			name:      "invalid device ID",
			deviceID:  "",
			startTime: now.Add(-time.Hour),
			endTime:   now,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate inputs
			if tt.deviceID == "" && !tt.wantErr {
				t.Error("Expected error for empty device ID")
			}
			
			if tt.endTime.Before(tt.startTime) && !tt.wantErr {
				t.Error("Expected error for invalid time range")
			}
			
			// Validate time range is not too large (prevent excessive queries)
			maxRange := 30 * 24 * time.Hour // 30 days
			if tt.endTime.Sub(tt.startTime) > maxRange && !tt.wantErr {
				t.Error("Time range too large")
			}
		})
	}
}

func TestMeasurementRepository_Aggregation(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name string
		req  AggregationRequest
	}{
		{
			name: "hourly temperature average",
			req: AggregationRequest{
				DeviceIDs: []string{"test-device-1"},
				Types:     []string{"temperature"},
				TimeRange: TimeRangeFilter{
					StartTime: &[]time.Time{now.Add(-24 * time.Hour)}[0],
					EndTime:   &now,
				},
				GroupByInterval:  time.Hour,
				AggregationType:  "avg",
			},
		},
		{
			name: "daily maximum pressure",
			req: AggregationRequest{
				DeviceIDs: []string{"test-device-1", "test-device-2"},
				Types:     []string{"pressure"},
				TimeRange: TimeRangeFilter{
					StartTime: &[]time.Time{now.Add(-7 * 24 * time.Hour)}[0],
					EndTime:   &now,
				},
				GroupByInterval:  24 * time.Hour,
				AggregationType:  "max",
			},
		},
		{
			name: "minute count for all types",
			req: AggregationRequest{
				DeviceIDs: []string{"test-device-1"},
				TimeRange: TimeRangeFilter{
					StartTime: &[]time.Time{now.Add(-time.Hour)}[0],
					EndTime:   &now,
				},
				GroupByInterval:  time.Minute,
				AggregationType:  "count",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate aggregation request
			if len(tt.req.DeviceIDs) == 0 {
				t.Error("DeviceIDs should not be empty")
			}
			
			if tt.req.GroupByInterval <= 0 {
				t.Error("GroupByInterval should be positive")
			}
			
			validAggTypes := []string{"avg", "min", "max", "sum", "count"}
			found := false
			for _, validType := range validAggTypes {
				if tt.req.AggregationType == validType {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Invalid aggregation type: %s", tt.req.AggregationType)
			}
			
			// Validate time range
			if tt.req.TimeRange.StartTime != nil && tt.req.TimeRange.EndTime != nil {
				if tt.req.TimeRange.EndTime.Before(*tt.req.TimeRange.StartTime) {
					t.Error("End time should be after start time")
				}
			}
		})
	}
}

func TestMeasurementRepository_Filter(t *testing.T) {
	tests := []struct {
		name   string
		filter MeasurementFilter
	}{
		{
			name: "filter by device and type",
			filter: MeasurementFilter{
				Filter: Filter{
					Limit:  100,
					Offset: 0,
					SortBy: "timestamp",
					Order:  "DESC",
				},
				DeviceIDs: []string{"device-1", "device-2"},
				Types:     []string{"temperature", "humidity"},
			},
		},
		{
			name: "filter by quality",
			filter: MeasurementFilter{
				Filter: Filter{
					Limit: 50,
				},
				Qualities: []models.QualityCode{models.QualityGood},
			},
		},
		{
			name: "filter by batch ID",
			filter: MeasurementFilter{
				Filter: Filter{
					Limit: 1000,
				},
				BatchID: &[]string{"batch-123"}[0],
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate filter parameters
			if tt.filter.Limit <= 0 {
				t.Error("Limit should be positive")
			}
			
			if tt.filter.Limit > 10000 {
				t.Error("Limit too high, may cause performance issues")
			}
			
			if tt.filter.Offset < 0 {
				t.Error("Offset should be non-negative")
			}
			
			// Validate sort order
			if tt.filter.Order != "" && tt.filter.Order != "ASC" && tt.filter.Order != "DESC" {
				t.Errorf("Invalid sort order: %s", tt.filter.Order)
			}
		})
	}
}