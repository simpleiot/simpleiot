package islog

import (
	"github.com/360EntSecGroup-Skylar/excelize/v2"
)

// ArrayToSpreadsheet takes a two-dimensional array of strings, data, and
// returns a type that can be saved in Microsoft Excel format
func ArrayToSpreadsheet(data [][]string) (*excelize.File, error) {

	// Create .xlsx file
	f := excelize.NewFile()

	for i, row := range data {
		for j, val := range row {
			cell, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				return nil, err
			}

			err = f.SetCellValue("Sheet1", cell, val)
			if err != nil {
				return nil, err
			}
		}
	}

	return f, nil
}
