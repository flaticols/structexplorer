package structexplorer

import "strings"

// JSON-safe data types for API responses (no template.HTML types)

type JSONIndexData struct {
	Rows []JSONTableRow `json:"rows"`
}

type JSONTableRow struct {
	Cells []JSONFieldList `json:"cells"`
}

type JSONFieldList struct {
	Label    string           `json:"label"`
	Path     string           `json:"path"`
	Row      int              `json:"row"`
	Column   int              `json:"column"`
	Type     string           `json:"type"`
	IsRoot   bool             `json:"is_root"`
	HasZeros bool             `json:"has_zeros"`
	Fields   []JSONFieldEntry `json:"fields"`
}

type JSONFieldEntry struct {
	Label string `json:"label"`
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ToJSON converts indexData to JSON-safe format
func (d indexData) ToJSON() JSONIndexData {
	result := JSONIndexData{
		Rows: make([]JSONTableRow, len(d.Rows)),
	}

	for i, row := range d.Rows {
		result.Rows[i] = JSONTableRow{
			Cells: make([]JSONFieldList, len(row.Cells)),
		}
		for j, cell := range row.Cells {
			// Strip HTML entities from label
			label := strings.ReplaceAll(string(cell.Label), "&nbsp;", "")
			label = strings.TrimSpace(label)

			result.Rows[i].Cells[j] = JSONFieldList{
				Label:    label,
				Path:     cell.Path,
				Row:      cell.Row,
				Column:   cell.Column,
				Type:     cell.Type,
				IsRoot:   cell.IsRoot,
				HasZeros: cell.HasZeros,
				Fields:   make([]JSONFieldEntry, len(cell.Fields)),
			}
			for k, field := range cell.Fields {
				result.Rows[i].Cells[j].Fields[k] = JSONFieldEntry{
					Label: field.Label,
					Key:   field.Key,
					Type:  field.Type,
					Value: field.ValueString,
				}
			}
		}
	}

	return result
}
