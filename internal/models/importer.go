package models

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ImportOptions struct {
	// If true, points absent in this import will be soft-deleted.
	// If false, they will be set Enabled=false.
	SoftDeleteMissing bool
}

func ParseSpec(data []byte) (TypeSpec, error) {
	var spec TypeSpec
	b := bytes.TrimSpace(data)
	if len(b) == 0 {
		return spec, fmt.Errorf("empty spec")
	}

	// JSON if starts with '{' or '['; otherwise YAML
	if b[0] == '{' || b[0] == '[' {
		if err := json.Unmarshal(b, &spec); err != nil {
			return spec, err
		}
		return spec, nil
	}
	if err := yaml.Unmarshal(b, &spec); err != nil {
		return spec, err
	}
	return spec, nil
}

func ValidateSpec(spec TypeSpec) error {
	if strings.TrimSpace(spec.TypeKey) == "" {
		return fmt.Errorf("type_key is required")
	}
	if strings.TrimSpace(spec.NameEn) == "" {
		return fmt.Errorf("name_en is required")
	}
	// vendor/model optional at type header level (you要求 points 表有；这里仍建议填)
	if len(spec.Points) == 0 {
		return fmt.Errorf("points is required")
	}

	seen := map[string]struct{}{}
	for i, p := range spec.Points {
		if strings.TrimSpace(p.Code) == "" {
			return fmt.Errorf("points[%d].code is required", i)
		}
		if _, ok := seen[p.Code]; ok {
			return fmt.Errorf("duplicate point code: %s", p.Code)
		}
		seen[p.Code] = struct{}{}

		if p.Kind != RegHolding && p.Kind != RegInput && p.Kind != RegCoil && p.Kind != RegDiscrete {
			return fmt.Errorf("points[%d].kind must be YC/YX/YK/YT", i)
		}
		// Requirement #9: English name required
		en := strings.TrimSpace(p.NameI18n["en"])
		if en == "" {
			return fmt.Errorf("points[%d].name_i18n.en is required", i)
		}
		// basic modbus validation
		if p.Modbus.FC == 0 {
			return fmt.Errorf("points[%d].modbus.fc is required", i)
		}
		if strings.TrimSpace(p.Modbus.DataType) == "" {
			return fmt.Errorf("points[%d].modbus.data_type is required", i)
		}
		if p.Modbus.Quantity == 0 {
			p.Modbus.Quantity = 1
		}
	}
	return nil
}

func ImportDeviceType(db *gorm.DB, spec TypeSpec, opt ImportOptions) error {
	if err := ValidateSpec(spec); err != nil {
		return err
	}

	// checksum of raw point list for audit (stable enough)
	raw, _ := json.Marshal(spec)
	sum := sha256.Sum256(raw)
	checksum := hex.EncodeToString(sum[:])

	return db.Transaction(func(tx *gorm.DB) error {
		// 1) Upsert device_types by type_key
		dt := DeviceType{
			ID:       uuid.NewString(),
			TypeKey:  spec.TypeKey,
			Name:     spec.NameEn,
			Vendor:   spec.Vendor,
			Model:    spec.Model,
			Version:  spec.Version,
			Checksum: checksum,
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "type_key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name_en", "vendor", "model", "version", "checksum", "updated_at",
			}),
		}).Create(&dt).Error; err != nil {
			return err
		}

		// 2) Upsert device_type_points by (type_key, point_code)
		codes := make([]string, 0, len(spec.Points))
		for _, p := range spec.Points {
			codes = append(codes, p.Code)

			enumJSON := []byte(`null`)
			if p.Modbus.EnumMap != nil {
				b, err := json.Marshal(p.Modbus.EnumMap)
				if err != nil {
					return fmt.Errorf("point %s enum_map marshal: %w", p.Code, err)
				}
				enumJSON = b
			}

			row := DeviceTypePoint{
				ID:        uuid.NewString(),
				TypeKey:   spec.TypeKey,
				PointCode: p.Code,
				PointKind: p.Kind,

				// Requirement #7: stored in points table
				Vendor: spec.Vendor,
				Model:  spec.Model,

				NameI18n: p.NameI18n,

				Unit:    p.Unit,
				RW:      normalizeRW(p.RW),
				Enabled: true,

				FC:        p.Modbus.FC,
				Address:   p.Modbus.Address,
				Quantity:  defaultU16(p.Modbus.Quantity, 1),
				DataType:  p.Modbus.DataType,
				BitIndex:  p.Modbus.BitIndex,
				ByteOrder: p.Modbus.ByteOrder,
				Scale:     defaultF64(p.Modbus.Scale, 1),
				Offset:    p.Modbus.Offset,
				Precision: p.Modbus.Precision,

				EnumMapJSON: enumJSON,
			}

			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "type_key"}, {Name: "point_code"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"point_kind",
					"vendor", "model",
					"name_i18n",
					"unit", "rw", "enabled",
					"fc", "address", "quantity", "data_type", "bit_index", "byte_order",
					"scale", "offset", "precision",
					"enum_map_json",
					"updated_at",
					"deleted_at", // important: revive if previously deleted
				}),
			}).Create(&row).Error; err != nil {
				return err
			}
		}

		// 3) Missing points handling
		if opt.SoftDeleteMissing {
			// soft-delete points not present in import
			if len(codes) > 0 {
				if err := tx.Where("type_key = ? AND point_code NOT IN ?", spec.TypeKey, codes).
					Delete(&DeviceTypePoint{}).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Where("type_key = ?", spec.TypeKey).
					Delete(&DeviceTypePoint{}).Error; err != nil {
					return err
				}
			}
		} else {
			// disable missing points
			if len(codes) > 0 {
				if err := tx.Model(&DeviceTypePoint{}).
					Where("type_key = ? AND point_code NOT IN ?", spec.TypeKey, codes).
					Updates(map[string]any{"enabled": false}).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Model(&DeviceTypePoint{}).
					Where("type_key = ?", spec.TypeKey).
					Updates(map[string]any{"enabled": false}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func normalizeRW(v string) string {
	s := strings.ToUpper(strings.TrimSpace(v))
	switch s {
	case "R", "W", "RW":
		return s
	default:
		return "R"
	}
}

func defaultU16(v uint16, def uint16) uint16 {
	if v == 0 {
		return def
	}
	return v
}

func defaultF64(v float64, def float64) float64 {
	// scale=0 usually means “not set”，默认给 1
	if v == 0 {
		return def
	}
	return v
}

/*
	data, err := os.ReadFile(os.Args[2])
	fatalIf(err)

	spec, err := importer.ParseSpec(data)
	fatalIf(err)

	opt := importer.ImportOptions{SoftDeleteMissing: true}
	fatalIf(importer.ImportDeviceType(gdb, spec, opt))

	fmt.Printf("import ok: type_key=%s points=%d\n", spec.TypeKey, len(spec.Points))
*/
