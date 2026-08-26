package helper

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

func GenerateFile(data [][]string, format types.ExportFormat) ([]byte, error) {
	if len(data) == 0 || len(data[0]) == 0 {
		return nil, errors.New("data is empty")
	}
	switch format {
	case types.ExportFormatCSV:
		return generateCSV(data)
	case types.ExportFormatPDF:
		return generatePDF(data)
	case types.ExportFormatExcel:
		return generateExcel(data)
	default:
		return generateImage(data)
	}
}

// GenerateCSV generates a CSV file from a 2D string array
// First row is treated as headers, subsequent rows as data
func generateCSV(data [][]string) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write all rows (headers + data)
	for _, row := range data {
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to flush writer: %w", err)
	}

	return buf.Bytes(), nil
}

// GeneratePDF generates a PDF document from a 2D string array
// First row is treated as headers, subsequent rows as data
func generatePDF(data [][]string) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	headers := data[0]
	rows := data[1:]

	// Use landscape orientation if there are many columns
	orientation := "P"
	if len(headers) > 6 {
		orientation = "L"
	}

	pdf := gofpdf.New(orientation, "mm", "A4", "")
	pdf.AddPage()

	// Calculate column width
	pageWidth, _ := pdf.GetPageSize()
	marginLeft, _, marginRight, _ := pdf.GetMargins()
	availableWidth := pageWidth - marginLeft - marginRight
	colWidth := availableWidth / float64(len(headers))

	// Use smaller font if columns are narrow
	headerFontSize := 8.0
	dataFontSize := 7.0
	if colWidth < 20 {
		headerFontSize = 7.0
		dataFontSize = 6.0
	}

	// Write headers
	pdf.SetFont("Arial", "B", headerFontSize)
	for _, header := range headers {
		// Only truncate header if column is narrow (< 15mm)
		headerText := header
		if colWidth < 15 {
			headerText = truncateString(header, colWidth, headerFontSize)
		}
		pdf.CellFormat(colWidth, 7, headerText, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Write data rows
	pdf.SetFont("Arial", "", dataFontSize)
	for _, row := range rows {
		for _, cell := range row {
			// Only truncate value if column is narrow (< 15mm)
			dataText := cell
			if colWidth < 15 {
				dataText = truncateString(cell, colWidth, dataFontSize)
			}
			pdf.CellFormat(colWidth, 6, dataText, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	// Return PDF as byte buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// truncateString truncates a string to fit within a given column width
// Approximates character width based on font size (rough estimate: 1 char ≈ fontSize * 0.35 mm)
func truncateString(s string, colWidth float64, fontSize float64) string {
	// More accurate estimate: average character width for Arial is about fontSize * 0.35 mm
	charWidth := fontSize * 0.35
	maxChars := int((colWidth - 1) / charWidth) // -1 for minimal cell padding

	if maxChars <= 0 {
		maxChars = 1
	}

	if len(s) <= maxChars {
		return s
	}

	if maxChars <= 3 {
		return s[:maxChars]
	}

	return s[:maxChars-3] + "..."
}

// GenerateImage generates a PNG image from a 2D string array
// First row is treated as headers, subsequent rows as data
func generateImage(data [][]string) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	headers := data[0]
	rows := data[1:]

	// Calculate image dimensions
	cellWidth := 120
	cellHeight := 30
	headerHeight := 40
	width := cellWidth * len(headers)
	height := headerHeight + (cellHeight * len(rows))

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background with white
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Draw header background
	headerColor := color.RGBA{70, 130, 180, 255} // Steel blue
	for y := 0; y < headerHeight; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, headerColor)
		}
	}

	// Draw grid lines
	lineColor := color.RGBA{0, 0, 0, 255} // Black

	// Horizontal lines
	for i := 0; i <= len(rows); i++ {
		y := headerHeight + (i * cellHeight)
		for x := 0; x < width; x++ {
			img.Set(x, y, lineColor)
		}
	}

	// Vertical lines
	for i := 0; i <= len(headers); i++ {
		x := i * cellWidth
		for y := 0; y < height; y++ {
			img.Set(x, y, lineColor)
		}
	}

	// Draw text (simplified - just drawing column names and basic text representation)
	// Note: For proper text rendering, you would typically use a library like golang.org/x/image/font
	// This is a simplified version that creates placeholder text areas

	// Add simple text markers for headers
	textColor := color.RGBA{255, 255, 255, 255} // White text on blue background
	for colIdx, header := range headers {
		x := colIdx*cellWidth + 5
		y := 15
		drawSimpleText(img, x, y, header, textColor)
	}

	// Add simple text markers for data rows
	dataTextColor := color.RGBA{0, 0, 0, 255} // Black text
	for rowIdx, row := range rows {
		for colIdx, cell := range row {
			x := colIdx*cellWidth + 5
			y := headerHeight + (rowIdx * cellHeight) + 15
			// Truncate long values
			value := cell
			if len(value) > 15 {
				value = value[:12] + "..."
			}
			drawSimpleText(img, x, y, value, dataTextColor)
		}
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateExcel generates an Excel (.xlsx) file from a 2D string array
// First row is treated as headers, subsequent rows as data
func generateExcel(data [][]string) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	// Create a new Excel file
	file := excelize.NewFile()
	defer func() {
		// if err := file.Close(); err != nil {
		// 	// Handle error if needed
		// }
	}()

	sheetName := "Sheet1"
	index, err := file.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet: %w", err)
	}
	file.SetActiveSheet(index)

	headers := data[0]
	rows := data[1:]

	// Create header style
	headerStyle, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4682B4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create header style: %w", err)
	}

	// Write headers
	for colIdx, header := range headers {
		cell, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name: %w", err)
		}
		if err := file.SetCellValue(sheetName, cell, header); err != nil {
			return nil, fmt.Errorf("failed to set header value: %w", err)
		}
		if err := file.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return nil, fmt.Errorf("failed to set header style: %w", err)
		}
	}

	// Write data rows
	for rowIdx, row := range rows {
		for colIdx, cell := range row {
			cellName, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				return nil, fmt.Errorf("failed to get cell name: %w", err)
			}
			if err := file.SetCellValue(sheetName, cellName, cell); err != nil {
				return nil, fmt.Errorf("failed to set cell value: %w", err)
			}
		}
	}

	// Auto-fit columns
	for colIdx := range headers {
		col, err := excelize.ColumnNumberToName(colIdx + 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get column name: %w", err)
		}
		if err := file.SetColWidth(sheetName, col, col, 15); err != nil {
			return nil, fmt.Errorf("failed to set column width: %w", err)
		}
	}

	// Save to buffer
	buf, err := file.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write to buffer: %w", err)
	}

	return buf.Bytes(), nil
}

// drawSimpleText is a helper function to draw simple text on an image
// This is a very basic implementation - for production use, consider using proper font rendering libraries
func drawSimpleText(img *image.RGBA, x, y int, text string, col color.Color) {
	// This is a placeholder that just marks the text position
	// In a real implementation, you would use a proper font rendering library
	// like golang.org/x/image/font and golang.org/x/image/font/basicfont

	// For now, we'll just draw a small indicator at the text position
	// to show where text would be rendered
	for i := 0; i < len(text) && i < 15; i++ {
		for dx := 0; dx < 5; dx++ {
			for dy := 0; dy < 7; dy++ {
				if x+(i*6)+dx < img.Bounds().Max.X && y+dy < img.Bounds().Max.Y {
					img.Set(x+(i*6)+dx, y+dy, col)
				}
			}
		}
	}
}
