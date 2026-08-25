package types

type ExportFormat string

const (
	ExportFormatPDF   ExportFormat = "PDF"
	ExportFormatCSV   ExportFormat = "CSV"
	ExportFormatExcel ExportFormat = "EXCEL"
	ExportFormatImage ExportFormat = "IMAGE"
)

var ExportFormats = []ExportFormat{
	ExportFormatPDF,
	ExportFormatCSV,
	ExportFormatExcel,
	ExportFormatImage,
}
