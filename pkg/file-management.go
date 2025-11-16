package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lms-tui/logger"
	"lms-tui/models"

	excelize "github.com/xuri/excelize/v2"
)

// JobData represents the structured data from an Excel job file
type JobData struct {
	JobNumber    string       `json:"job_number"`
	ProjectName  string       `json:"project_name"`
	Engineer     string       `json:"engineer"`
	Date         string       `json:"date"`
	DueDate      string       `json:"due_date"`
	PageInfo     string       `json:"page_info"`
	TotalSamples int          `json:"total_samples"`
	Samples      []SampleData `json:"samples"`
}

// SampleData represents a single sample/boring entry
type SampleData struct {
	BoringNumber         string   `json:"boring_number"`
	Depth                string   `json:"depth"`
	Tests                []string `json:"tests"`
}

// ExcelToJSON converts Excel data to JSON format and logs it
func ExcelToJSON(filePath string) (*JobData, error) {
	f, err := excelize.OpenFile(GetProjectPath(filePath))
	if err != nil {
		logger.Error.Printf("Failed to open Excel file for JSON conversion: %v", err)
		return nil, err
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		logger.Error.Printf("Failed to read rows: %v", err)
		return nil, err
	}

	jobData := &JobData{
		Samples: []SampleData{},
	}

	// Parse the header information
	for rowIdx, row := range rows {
		if len(row) == 0 {
			continue
		}

		// Look for specific labels in the first column
		firstCell := strings.TrimSpace(row[0])

		switch {
		case firstCell == "Job No." && len(row) > 2:
			jobData.JobNumber = strings.TrimSpace(row[2])
		case firstCell == "Project Name." && len(row) > 2:
			jobData.ProjectName = strings.TrimSpace(row[2])
			if len(row) > 5 {
				jobData.Engineer = strings.TrimSpace(row[5])
			}
			if len(row) > 9 {
				jobData.Date = strings.TrimSpace(row[9])
			}
		case strings.Contains(firstCell, "Due Date") && len(row) > 9:
			jobData.DueDate = strings.TrimSpace(row[9])
		case strings.HasPrefix(firstCell, "B-") || (rowIdx > 6 && firstCell == ""):
			// This is a sample row
			if len(row) > 1 {
				sample := SampleData{
					Tests: []string{},
				}

				// Check if this is a new boring or continuation
				if strings.HasPrefix(firstCell, "B-") {
					sample.BoringNumber = firstCell
				}

				// Get depth
				if len(row) > 1 && strings.TrimSpace(row[1]) != "" {
					sample.Depth = strings.TrimSpace(row[1])
				}

				// Check for test markers (x's in various columns)
				testNames := []string{
					"Atterberg Limit",
					"Atterberg Limit (w/ lime)",
					"Moisture Content",
					"Absorption Pressure Swell",
					"QU",
					"Gradation",
					"Soil Suction",
				}
				testCols := []int{2, 3, 4, 5, 6, 7, 9} // Approximate column indices

				for i, col := range testCols {
					if col < len(row) && strings.TrimSpace(row[col]) == "x" {
						if i < len(testNames) {
							sample.Tests = append(sample.Tests, testNames[i])
						}
					}
				}

				// Only add if we have a depth (valid sample)
				if sample.Depth != "" {
					jobData.Samples = append(jobData.Samples, sample)
					jobData.TotalSamples++
				}
			}
		}
	}

	// Assign boring numbers to samples that don't have them (they belong to the previous boring)
	currentBoring := ""
	for i := range jobData.Samples {
		if jobData.Samples[i].BoringNumber != "" {
			currentBoring = jobData.Samples[i].BoringNumber
		} else {
			jobData.Samples[i].BoringNumber = currentBoring
		}
	}

	// Convert to JSON and log
	jsonBytes, err := json.MarshalIndent(jobData, "", "  ")
	if err != nil {
		logger.Error.Printf("Failed to convert to JSON: %v", err)
		return nil, err
	}

	logger.Info.Printf("Excel data converted to JSON:\n%s", string(jsonBytes))

	return jobData, nil
}

// ProjectRoot is the root directory of the project
const ProjectRoot = "/home/marcomascorro/developer/Reed-Engineering/lms"

// GetProjectPath returns the full path relative to the project root
func GetProjectPath(relativePath string) string {
	filepath := filepath.Join(ProjectRoot, relativePath)

	logger.Info.Printf("Project path: %s", filepath)
	return filepath
}

func GetExcelFile(filePath string) (*excelize.File, error) {
	path := GetProjectPath(filePath)
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// LogExcelData reads an Excel file and logs its contents in a formatted table
func LogExcelData(filePath string) error {
	// Open the Excel file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		logger.Error.Printf("Failed to open Excel file: %s - %v", filePath, err)
		return err
	}
	defer f.Close()

	// Get the first sheet name
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		logger.Error.Println("No sheets found in Excel file")
		return fmt.Errorf("no sheets found")
	}

	// Get all rows from the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		logger.Error.Printf("Failed to read rows: %v", err)
		return err
	}

	if len(rows) == 0 {
		logger.Info.Println("Excel file is empty")
		return nil
	}

	// Calculate max width for each column
	colWidths := make([]int, 0)
	for _, row := range rows {
		for colIdx, cell := range row {
			if colIdx >= len(colWidths) {
				colWidths = append(colWidths, 0)
			}
			if len(cell) > colWidths[colIdx] {
				colWidths[colIdx] = len(cell)
			}
		}
	}

	// Set minimum width and cap maximum
	for i := range colWidths {
		if colWidths[i] < 10 {
			colWidths[i] = 10
		}
		if colWidths[i] > 30 {
			colWidths[i] = 30
		}
	}

	// Create separator line
	separatorParts := make([]string, len(colWidths))
	for i, width := range colWidths {
		separatorParts[i] = strings.Repeat("-", width)
	}
	separator := "+-" + strings.Join(separatorParts, "-+-") + "-+"

	// Log header
	logger.Info.Printf("Excel Data from: %s (Sheet: %s)", filePath, sheetName)
	logger.Info.Println(separator)

	// Log each row
	for rowIdx, row := range rows {
		// Build formatted row
		cellParts := make([]string, len(colWidths))
		for colIdx := range colWidths {
			value := ""
			if colIdx < len(row) {
				value = row[colIdx]
				// Truncate if too long
				if len(value) > colWidths[colIdx] {
					value = value[:colWidths[colIdx]-3] + "..."
				}
			}
			cellParts[colIdx] = fmt.Sprintf("%-*s", colWidths[colIdx], value)
		}
		logger.Info.Printf("| %s |", strings.Join(cellParts, " | "))

		// Add separator after header row
		if rowIdx == 0 {
			logger.Info.Println(separator)
		}
	}

	logger.Info.Println(separator)
	logger.Info.Printf("Total rows: %d", len(rows))

	return nil
}

// MoistureTestWriter manages writing moisture test data to Excel
type MoistureTestWriter struct {
	JobNumber    string
	FilePath     string
	file         *excelize.File
	sampleColMap map[string]string // Maps "BoringNo|Depth" to "SheetName|ColumnLetter"
}

// InitMoistureTestFile creates the ex_project directory, copies the Lab file, and initializes the moisture writer
func InitMoistureTestFile(jobNumber string) (*MoistureTestWriter, error) {
	// Create directory structure
	dirPath := filepath.Join(ProjectRoot, "ex_project", jobNumber)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		logger.Error.Printf("Failed to create directory %s: %v", dirPath, err)
		return nil, err
	}
	logger.Info.Printf("Created/verified directory: %s", dirPath)

	// Source and destination paths
	srcPath := filepath.Join(ProjectRoot, "projects", jobNumber, fmt.Sprintf("Lab_%s.xlsm", jobNumber))
	dstPath := filepath.Join(dirPath, fmt.Sprintf("Lab_%s.xlsm", jobNumber))

	writer := &MoistureTestWriter{
		JobNumber:    jobNumber,
		FilePath:     dstPath,
		sampleColMap: make(map[string]string),
	}

	// Check if destination file exists, if not copy from source
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		// Copy the source file
		srcData, err := os.ReadFile(srcPath)
		if err != nil {
			logger.Error.Printf("Failed to read source Lab file: %v", err)
			return nil, err
		}
		if err := os.WriteFile(dstPath, srcData, 0644); err != nil {
			logger.Error.Printf("Failed to copy Lab file to ex_project: %v", err)
			return nil, err
		}
		logger.Info.Printf("Copied Lab file to: %s", dstPath)
	}

	// Open the file
	var err error
	writer.file, err = excelize.OpenFile(dstPath)
	if err != nil {
		logger.Error.Printf("Failed to open Lab file: %v", err)
		return nil, err
	}

	// Build sample column map from all Moisture sheets (Moisture, Moisture2, Moisture3, etc.)
	// Row 9 has Boring No, Row 10 has Depth
	// Columns B onwards contain the sample data
	sheetNames := writer.file.GetSheetList()
	for _, sheetName := range sheetNames {
		// Check if this is a Moisture sheet
		if sheetName == "Moisture" || strings.HasPrefix(sheetName, "Moisture") && !strings.Contains(sheetName, " ") {
			rows, err := writer.file.GetRows(sheetName)
			if err != nil {
				logger.Error.Printf("Failed to read %s sheet: %v", sheetName, err)
				continue
			}

			if len(rows) >= 10 {
				boringRow := rows[8]  // Row 9 (0-indexed = 8)
				depthRow := rows[9]   // Row 10 (0-indexed = 9)

				// Map each column to its boring/depth combination
				for colIdx := 1; colIdx < len(boringRow) && colIdx < len(depthRow); colIdx++ {
					boring := strings.TrimSpace(boringRow[colIdx])
					depth := strings.TrimSpace(depthRow[colIdx])
					if boring != "" && depth != "" {
						colLetter := getColumnLetter(colIdx + 1) // +1 because Excel is 1-indexed
						key := fmt.Sprintf("%s|%s", boring, depth)
						// Store sheet name with column letter
						writer.sampleColMap[key] = fmt.Sprintf("%s|%s", sheetName, colLetter)
						logger.Info.Printf("Mapped sample %s to %s column %s", key, sheetName, colLetter)
					}
				}
			}
		}
	}

	logger.Info.Printf("Initialized moisture writer with %d sample mappings across multiple sheets", len(writer.sampleColMap))
	return writer, nil
}

// getColumnLetter converts a 1-based column index to Excel column letter (1=A, 2=B, etc.)
func getColumnLetter(colIdx int) string {
	result := ""
	for colIdx > 0 {
		colIdx--
		result = string(rune('A'+colIdx%26)) + result
		colIdx /= 26
	}
	return result
}

// WriteMoistureSample writes a single sample's moisture data to the appropriate Moisture sheet
func (w *MoistureTestWriter) WriteMoistureSample(boringNumber, depth, canNo, canWeight, wetWeight string) error {
	// Find the sheet and column for this sample
	key := fmt.Sprintf("%s|%s", boringNumber, depth)
	mapping, exists := w.sampleColMap[key]
	if !exists {
		logger.Error.Printf("No column mapping found for sample %s", key)
		return fmt.Errorf("no column mapping for %s", key)
	}

	// Parse sheet name and column letter from mapping (format: "SheetName|ColumnLetter")
	parts := strings.Split(mapping, "|")
	if len(parts) != 2 {
		logger.Error.Printf("Invalid mapping format for sample %s: %s", key, mapping)
		return fmt.Errorf("invalid mapping format for %s", key)
	}
	sheetName := parts[0]
	colLetter := parts[1]

	// Write data to the correct cells in the Moisture sheet
	// Row 11: Can No.
	// Row 12: Wet wt. and can
	// Row 15: Wt. of can (Can Weight)
	w.file.SetCellValue(sheetName, fmt.Sprintf("%s11", colLetter), canNo)
	w.file.SetCellValue(sheetName, fmt.Sprintf("%s12", colLetter), wetWeight)
	w.file.SetCellValue(sheetName, fmt.Sprintf("%s15", colLetter), canWeight)

	// Save file
	if err := w.file.Save(); err != nil {
		logger.Error.Printf("Failed to save moisture data: %v", err)
		return err
	}

	logger.Info.Printf("Wrote moisture sample to %s column %s: Boring=%s, Depth=%s, Can#=%s, CanWt=%s, WetWt=%s",
		sheetName, colLetter, boringNumber, depth, canNo, canWeight, wetWeight)

	return nil
}

// Close closes the Excel file
func (w *MoistureTestWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// GetFile returns the Excel file handle for sharing with other writers
func (w *MoistureTestWriter) GetFile() *excelize.File {
	return w.file
}

// GetSampleMapping returns the sheet name and column letter for a given boring/depth
func (w *MoistureTestWriter) GetSampleMapping(boringNumber, depth string) (string, string, bool) {
	key := fmt.Sprintf("%s|%s", boringNumber, depth)
	mapping, exists := w.sampleColMap[key]
	if !exists {
		return "", "", false
	}

	// Parse sheet name and column letter from mapping (format: "SheetName|ColumnLetter")
	parts := strings.Split(mapping, "|")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ProgressData represents saved progress for a job
type ProgressData struct {
	JobNumber          string `json:"job_number"`
	CurrentSampleIndex int    `json:"current_sample_index"`
	LastSaved          string `json:"last_saved"`
}

// SampleBackupData represents a single sample's backup data
type SampleBackupData struct {
	JobNumber     string `json:"job_number"`
	BoringNumber  string `json:"boring_number"`
	Depth         string `json:"depth"`
	CanNumber     string `json:"can_number"`
	CanWeight     string `json:"can_weight"`
	WetWeight     string `json:"wet_weight"`
	SuctionCanNo  string `json:"suction_can_no"`
	Timestamp     string `json:"timestamp"`
}

// BackupData represents the complete backup file structure
type BackupData struct {
	JobNumber    string             `json:"job_number"`
	LastUpdated  string             `json:"last_updated"`
	TotalSamples int                `json:"total_samples"`
	Samples      []SampleBackupData `json:"samples"`
}

// SaveSampleBackup saves a sample to the JSON backup file
func SaveSampleBackup(jobNumber, boringNumber, depth, canNo, canWeight, wetWeight, suctionCanNo string) error {
	dirPath := filepath.Join(ProjectRoot, "ex_project", jobNumber)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		logger.Error.Printf("Failed to create directory for backup: %v", err)
		return err
	}

	backupFile := filepath.Join(dirPath, "backup.json")

	// Load existing backup or create new one
	var backup BackupData
	if data, err := os.ReadFile(backupFile); err == nil {
		if err := json.Unmarshal(data, &backup); err != nil {
			logger.Error.Printf("Failed to unmarshal existing backup: %v", err)
			backup = BackupData{
				JobNumber: jobNumber,
				Samples:   []SampleBackupData{},
			}
		}
	} else {
		backup = BackupData{
			JobNumber: jobNumber,
			Samples:   []SampleBackupData{},
		}
	}

	// Create new sample entry
	newSample := SampleBackupData{
		JobNumber:    jobNumber,
		BoringNumber: boringNumber,
		Depth:        depth,
		CanNumber:    canNo,
		CanWeight:    canWeight,
		WetWeight:    wetWeight,
		SuctionCanNo: suctionCanNo,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	// Append to samples array
	backup.Samples = append(backup.Samples, newSample)
	backup.TotalSamples = len(backup.Samples)
	backup.LastUpdated = time.Now().Format("2006-01-02 15:04:05")

	// Save to file
	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		logger.Error.Printf("Failed to marshal backup data: %v", err)
		return err
	}

	if err := os.WriteFile(backupFile, jsonData, 0644); err != nil {
		logger.Error.Printf("Failed to write backup file: %v", err)
		return err
	}

	logger.Info.Printf("Saved sample backup: Job=%s, Boring=%s, Depth=%s", jobNumber, boringNumber, depth)
	return nil
}

// SaveProgress saves the current sample index to a progress file
func SaveProgress(jobNumber string, currentSampleIndex int) error {
	dirPath := filepath.Join(ProjectRoot, "ex_project", jobNumber)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		logger.Error.Printf("Failed to create directory for progress: %v", err)
		return err
	}

	progressFile := filepath.Join(dirPath, "progress.json")
	progress := ProgressData{
		JobNumber:          jobNumber,
		CurrentSampleIndex: currentSampleIndex,
		LastSaved:          fmt.Sprintf("%v", os.Getenv("USER")),
	}

	jsonData, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		logger.Error.Printf("Failed to marshal progress data: %v", err)
		return err
	}

	if err := os.WriteFile(progressFile, jsonData, 0644); err != nil {
		logger.Error.Printf("Failed to write progress file: %v", err)
		return err
	}

	logger.Info.Printf("Saved progress for job %s: sample index %d", jobNumber, currentSampleIndex)
	return nil
}

// LoadProgress loads the saved progress for a job
func LoadProgress(jobNumber string) (int, error) {
	progressFile := filepath.Join(ProjectRoot, "ex_project", jobNumber, "progress.json")

	data, err := os.ReadFile(progressFile)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info.Printf("No progress file found for job %s, starting from beginning", jobNumber)
			return 0, nil
		}
		logger.Error.Printf("Failed to read progress file: %v", err)
		return 0, err
	}

	var progress ProgressData
	if err := json.Unmarshal(data, &progress); err != nil {
		logger.Error.Printf("Failed to unmarshal progress data: %v", err)
		return 0, err
	}

	logger.Info.Printf("Loaded progress for job %s: resuming at sample index %d", jobNumber, progress.CurrentSampleIndex)
	return progress.CurrentSampleIndex, nil
}

// DiscoverJobs scans the projects folder for Lab_*.xlsm files and returns job information
func DiscoverJobs() ([]models.Job, error) {
	projectsDir := filepath.Join(ProjectRoot, "projects")
	var jobs []models.Job

	// Check if projects directory exists
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		logger.Info.Printf("Projects directory does not exist: %s", projectsDir)
		return jobs, nil
	}

	// Read all directories in the projects folder
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		logger.Error.Printf("Failed to read projects directory: %v", err)
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		jobNumber := entry.Name()
		labFilePath := filepath.Join(projectsDir, jobNumber, fmt.Sprintf("Lab_%s.xlsm", jobNumber))

		// Check if Lab file exists
		if _, err := os.Stat(labFilePath); os.IsNotExist(err) {
			continue
		}

		// Extract job info from Excel file
		job, err := extractJobInfoFromExcel(labFilePath, jobNumber)
		if err != nil {
			logger.Error.Printf("Failed to extract job info from %s: %v", labFilePath, err)
			continue
		}

		jobs = append(jobs, job)
		logger.Info.Printf("Discovered job: %s - %s", job.ProjectNumber, job.ProjectName)
	}

	logger.Info.Printf("Discovered %d jobs in projects folder", len(jobs))
	return jobs, nil
}

// extractJobInfoFromExcel reads job information from the Excel file
func extractJobInfoFromExcel(filePath string, jobNumber string) (models.Job, error) {
	job := models.Job{
		ProjectNumber:    jobNumber,
		ProjectName:      "Unknown Project",
		EngineerInitials: "N/A",
		DateAssigned:     time.Now(),
		DueDate:          time.Now().AddDate(0, 0, 14), // Default 14 days from now
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return job, err
	}
	defer f.Close()

	// Try to find the "Main Form" or "!Main Form" sheet first, otherwise use first sheet
	sheetName := ""
	sheetList := f.GetSheetList()
	for _, name := range sheetList {
		if name == "Main Form" || name == "!Main Form" {
			sheetName = name
			break
		}
	}
	if sheetName == "" {
		sheetName = f.GetSheetName(0)
		logger.Info.Printf("Main Form sheet not found, using first sheet: %s", sheetName)
	}

	// Read specific cells from the Main Form sheet
	// Row 4: Project Name in C4, Engineer in F4 or after "Engineer.", Date at end
	// Row 3: Job No. in C3
	// Row 5: Due Date at end

	// Try to find project name - search rows for "Project Name."
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return job, err
	}

	for rowIdx, row := range rows {
		if len(row) == 0 {
			continue
		}

		rowText := strings.Join(row, " ")

		// Look for Project Name row
		if strings.Contains(rowText, "Project Name.") {
			// Get Project Name from cell C of this row (rowIdx is 0-based, Excel is 1-based)
			excelRow := rowIdx + 1
			projectName, _ := f.GetCellValue(sheetName, fmt.Sprintf("C%d", excelRow))
			if strings.TrimSpace(projectName) != "" {
				job.ProjectName = strings.TrimSpace(projectName)
			}

			// Parse the row text for Engineer initials (after "Engineer.")
			if idx := strings.Index(rowText, "Engineer."); idx != -1 {
				afterEngineer := rowText[idx+9:]
				parts := strings.Fields(afterEngineer)
				if len(parts) > 0 {
					job.EngineerInitials = parts[0]
				}
			}

			// Parse the row text for Date (after "Date")
			if idx := strings.Index(rowText, "Date"); idx != -1 {
				afterDate := rowText[idx+4:]
				parts := strings.Fields(afterDate)
				if len(parts) > 0 {
					dateStr := parts[0]
					if parsedDate, err := parseExcelDate(dateStr); err == nil {
						job.DateAssigned = parsedDate
					}
				}
			}
		}

		// Look for Due Date row
		if strings.Contains(rowText, "Due Date") {
			// Parse the row text for due date
			if idx := strings.Index(rowText, "Due Date"); idx != -1 {
				afterDueDate := rowText[idx+8:]
				parts := strings.Fields(afterDueDate)
				if len(parts) > 0 {
					dateStr := parts[0]
					if parsedDate, err := parseExcelDate(dateStr); err == nil {
						job.DueDate = parsedDate
					}
				}
			}
		}
	}

	return job, nil
}

// parseExcelDate attempts to parse various date formats from Excel
func parseExcelDate(dateStr string) (time.Time, error) {
	// Try various date formats
	formats := []string{
		"01/02/2006",
		"1/2/2006",
		"2006-01-02",
		"01-02-2006",
		"Jan 2, 2006",
		"January 2, 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// SoilSuctionWriter manages writing soil suction test data to Excel
type SoilSuctionWriter struct {
	JobNumber        string
	FilePath         string
	file             *excelize.File
	sampleRowMap     map[string]string // Maps "BoringNo|Depth" to "SheetName|RowNumber"
	separateFile     *excelize.File    // Separate suction file
	separatePath     string            // Path to separate suction file
	separateNextRow  int               // Next row in separate file
	separateSheetNum int               // Current sheet number (1 = "Soil Suction", 2 = "Soil Suction 2", etc.)
}

// InitSoilSuctionFile initializes the soil suction writer using the same file handle as moisture writer
func InitSoilSuctionFile(jobNumber string, sharedFile *excelize.File) (*SoilSuctionWriter, error) {
	// The Lab file should already be copied by InitMoistureTestFile
	dirPath := filepath.Join(ProjectRoot, "ex_project", jobNumber)
	filePath := filepath.Join(dirPath, fmt.Sprintf("Lab_%s.xlsm", jobNumber))
	separatePath := filepath.Join(dirPath, fmt.Sprintf("SoilSuction_%s.xlsx", jobNumber))

	writer := &SoilSuctionWriter{
		JobNumber:        jobNumber,
		FilePath:         filePath,
		sampleRowMap:     make(map[string]string),
		file:             sharedFile, // Use shared file handle
		separatePath:     separatePath,
		separateNextRow:  2, // Start after header
		separateSheetNum: 1, // First sheet
	}

	// Create or open separate soil suction file
	if _, err := os.Stat(separatePath); os.IsNotExist(err) {
		// Create new file with headers
		writer.separateFile = excelize.NewFile()
		sheetName := "Soil Suction"
		writer.separateFile.SetSheetName("Sheet1", sheetName)

		// Set headers: Date, Boring, Depth, Can No, Top, Bottom, Top, Bottom
		writer.separateFile.SetCellValue(sheetName, "A1", "Date")
		writer.separateFile.SetCellValue(sheetName, "B1", "Boring")
		writer.separateFile.SetCellValue(sheetName, "C1", "Depth")
		writer.separateFile.SetCellValue(sheetName, "D1", "Can No")
		writer.separateFile.SetCellValue(sheetName, "E1", "Top")
		writer.separateFile.SetCellValue(sheetName, "F1", "Bottom")
		writer.separateFile.SetCellValue(sheetName, "G1", "Top")
		writer.separateFile.SetCellValue(sheetName, "H1", "Bottom")

		// Style headers
		style, _ := writer.separateFile.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		})
		writer.separateFile.SetCellStyle(sheetName, "A1", "H1", style)

		// Set column widths
		writer.separateFile.SetColWidth(sheetName, "A", "A", 12)
		writer.separateFile.SetColWidth(sheetName, "B", "B", 12)
		writer.separateFile.SetColWidth(sheetName, "C", "C", 12)
		writer.separateFile.SetColWidth(sheetName, "D", "D", 12)
		writer.separateFile.SetColWidth(sheetName, "E", "E", 12)
		writer.separateFile.SetColWidth(sheetName, "F", "F", 12)
		writer.separateFile.SetColWidth(sheetName, "G", "G", 12)
		writer.separateFile.SetColWidth(sheetName, "H", "H", 12)

		if err := writer.separateFile.SaveAs(separatePath); err != nil {
			logger.Error.Printf("Failed to create separate soil suction Excel file: %v", err)
			return nil, err
		}
		logger.Info.Printf("Created separate soil suction Excel file: %s", separatePath)
	} else {
		// Open existing file and find next empty row and current sheet
		var err error
		writer.separateFile, err = excelize.OpenFile(separatePath)
		if err != nil {
			logger.Error.Printf("Failed to open existing separate soil suction file: %v", err)
			return nil, err
		}

		// Find the last sheet and its next empty row
		sheetList := writer.separateFile.GetSheetList()
		writer.separateSheetNum = len(sheetList)

		// Get the current sheet name
		currentSheetName := "Soil Suction"
		if writer.separateSheetNum > 1 {
			currentSheetName = fmt.Sprintf("Soil Suction %d", writer.separateSheetNum)
		}

		rows, _ := writer.separateFile.GetRows(currentSheetName)
		writer.separateNextRow = len(rows) + 1

		// Check if current sheet is full (37 samples + 1 header = 38 rows)
		if writer.separateNextRow > 38 {
			// Need to create a new sheet
			writer.separateSheetNum++
			writer.separateNextRow = 2
		}

		logger.Info.Printf("Opened existing separate soil suction file, sheet %d, next row: %d", writer.separateSheetNum, writer.separateNextRow)
	}

	// Build sample row map from all Soil Suction sheets (Soil Suction, Soil Suction2, etc.)
	// Column B has Boring No., Column C has Depth
	// Starting from row 10
	sheetNames := writer.file.GetSheetList()
	for _, sheetName := range sheetNames {
		// Check if this is a Soil Suction sheet
		if sheetName == "Soil Suction" || strings.HasPrefix(sheetName, "Soil Suction") {
			rows, err := writer.file.GetRows(sheetName)
			if err != nil {
				logger.Error.Printf("Failed to read %s sheet: %v", sheetName, err)
				continue
			}

			// Map each row to its boring/depth combination (starting from row 10, index 9)
			for rowIdx := 9; rowIdx < len(rows); rowIdx++ {
				row := rows[rowIdx]
				if len(row) >= 3 {
					boring := strings.TrimSpace(row[1]) // Column B (index 1)
					depth := strings.TrimSpace(row[2])  // Column C (index 2)
					if boring != "" && depth != "" {
						key := fmt.Sprintf("%s|%s", boring, depth)
						actualRow := rowIdx + 1 // Convert to 1-based Excel row number
						// Store sheet name with row number
						writer.sampleRowMap[key] = fmt.Sprintf("%s|%d", sheetName, actualRow)
						logger.Info.Printf("Mapped soil suction sample %s to %s row %d", key, sheetName, actualRow)
					}
				}
			}
		}
	}

	logger.Info.Printf("Initialized soil suction writer with %d sample mappings across multiple sheets", len(writer.sampleRowMap))
	return writer, nil
}

// WriteSoilSuctionSample writes a single sample's soil suction can number to the appropriate Soil Suction sheet
func (w *SoilSuctionWriter) WriteSoilSuctionSample(boringNumber, depth, suctionCanNo string) error {
	// Find the sheet and row for this sample
	key := fmt.Sprintf("%s|%s", boringNumber, depth)
	mapping, exists := w.sampleRowMap[key]
	if !exists {
		logger.Error.Printf("No row mapping found for soil suction sample %s", key)
		return fmt.Errorf("no row mapping for %s", key)
	}

	// Parse sheet name and row number from mapping (format: "SheetName|RowNumber")
	parts := strings.Split(mapping, "|")
	if len(parts) != 2 {
		logger.Error.Printf("Invalid mapping format for soil suction sample %s: %s", key, mapping)
		return fmt.Errorf("invalid mapping format for %s", key)
	}
	sheetName := parts[0]
	rowNum := parts[1]

	// Write can number to column D of the correct row in Lab file
	w.file.SetCellValue(sheetName, fmt.Sprintf("D%s", rowNum), suctionCanNo)

	// Save Lab file
	if err := w.file.Save(); err != nil {
		logger.Error.Printf("Failed to save soil suction data to Lab file: %v", err)
		return err
	}

	// Also write to separate soil suction file
	if w.separateFile != nil {
		// Check if we need to create a new sheet (37 samples per sheet + 1 header = 38 rows max)
		if w.separateNextRow > 38 {
			// Create new sheet
			w.separateSheetNum++
			newSheetName := fmt.Sprintf("Soil Suction %d", w.separateSheetNum)
			w.separateFile.NewSheet(newSheetName)

			// Set headers for new sheet
			w.separateFile.SetCellValue(newSheetName, "A1", "Date")
			w.separateFile.SetCellValue(newSheetName, "B1", "Boring")
			w.separateFile.SetCellValue(newSheetName, "C1", "Depth")
			w.separateFile.SetCellValue(newSheetName, "D1", "Can No")
			w.separateFile.SetCellValue(newSheetName, "E1", "Top")
			w.separateFile.SetCellValue(newSheetName, "F1", "Bottom")
			w.separateFile.SetCellValue(newSheetName, "G1", "Top")
			w.separateFile.SetCellValue(newSheetName, "H1", "Bottom")

			// Style headers
			style, _ := w.separateFile.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true},
				Fill: excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
			})
			w.separateFile.SetCellStyle(newSheetName, "A1", "H1", style)

			// Set column widths
			w.separateFile.SetColWidth(newSheetName, "A", "A", 12)
			w.separateFile.SetColWidth(newSheetName, "B", "B", 12)
			w.separateFile.SetColWidth(newSheetName, "C", "C", 12)
			w.separateFile.SetColWidth(newSheetName, "D", "D", 12)
			w.separateFile.SetColWidth(newSheetName, "E", "E", 12)
			w.separateFile.SetColWidth(newSheetName, "F", "F", 12)
			w.separateFile.SetColWidth(newSheetName, "G", "G", 12)
			w.separateFile.SetColWidth(newSheetName, "H", "H", 12)

			w.separateNextRow = 2
			logger.Info.Printf("Created new sheet '%s' in separate soil suction file", newSheetName)
		}

		// Get current sheet name
		separateSheet := "Soil Suction"
		if w.separateSheetNum > 1 {
			separateSheet = fmt.Sprintf("Soil Suction %d", w.separateSheetNum)
		}

		currentDate := time.Now().Format("01/02/2006")

		// Write data: Date, Boring, Depth, Can No, Top (blank), Bottom (blank), Top (blank), Bottom (blank)
		w.separateFile.SetCellValue(separateSheet, fmt.Sprintf("A%d", w.separateNextRow), currentDate)
		w.separateFile.SetCellValue(separateSheet, fmt.Sprintf("B%d", w.separateNextRow), boringNumber)
		w.separateFile.SetCellValue(separateSheet, fmt.Sprintf("C%d", w.separateNextRow), depth)
		w.separateFile.SetCellValue(separateSheet, fmt.Sprintf("D%d", w.separateNextRow), suctionCanNo)
		// Columns E, F, G, H are left blank for Top/Bottom values

		// Save separate file
		if err := w.separateFile.Save(); err != nil {
			logger.Error.Printf("Failed to save separate soil suction file: %v", err)
			return err
		}

		logger.Info.Printf("Wrote soil suction to separate file sheet '%s' row %d", separateSheet, w.separateNextRow)
		w.separateNextRow++
	}

	logger.Info.Printf("Wrote soil suction can number to %s row %s (D%s): Boring=%s, Depth=%s, SuctionCan#=%s",
		sheetName, rowNum, rowNum, boringNumber, depth, suctionCanNo)

	return nil
}

// Close closes the Excel file
func (w *SoilSuctionWriter) Close() error {
	// Close separate file if it exists
	if w.separateFile != nil {
		w.separateFile.Close()
	}
	// Note: Don't close w.file here as it's shared with MoistureTestWriter
	return nil
}

// OvenCanData represents a moisture can currently in the oven
type OvenCanData struct {
	CanNumber       string `json:"can_number"`
	JobNumber       string `json:"job_number"`
	BoringNumber    string `json:"boring_number"`
	Depth           string `json:"depth"`
	TimeIn          string `json:"time_in"`
	MoistureSheet   string `json:"moisture_sheet"`   // Sheet name (e.g., "Moisture", "Moisture2")
	MoistureColumn  string `json:"moisture_column"`  // Column letter (e.g., "B", "C")
}

// OvenTrackingData represents all cans currently in the oven
type OvenTrackingData struct {
	Cans        []OvenCanData `json:"cans"`
	LastUpdated string        `json:"last_updated"`
}

// GetOvenTrackingFilePath returns the path to the global oven tracking file
func GetOvenTrackingFilePath() string {
	return filepath.Join(ProjectRoot, "oven_tracking.json")
}

// LoadOvenTracking loads the current oven tracking data
func LoadOvenTracking() (*OvenTrackingData, error) {
	filePath := GetOvenTrackingFilePath()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty tracking data if file doesn't exist
			return &OvenTrackingData{
				Cans:        []OvenCanData{},
				LastUpdated: time.Now().Format("2006-01-02 15:04:05"),
			}, nil
		}
		logger.Error.Printf("Failed to read oven tracking file: %v", err)
		return nil, err
	}

	var tracking OvenTrackingData
	if err := json.Unmarshal(data, &tracking); err != nil {
		logger.Error.Printf("Failed to unmarshal oven tracking data: %v", err)
		return nil, err
	}

	logger.Info.Printf("Loaded oven tracking data: %d cans in oven", len(tracking.Cans))
	return &tracking, nil
}

// SaveOvenTracking saves the oven tracking data to disk
func SaveOvenTracking(tracking *OvenTrackingData) error {
	filePath := GetOvenTrackingFilePath()

	tracking.LastUpdated = time.Now().Format("2006-01-02 15:04:05")

	jsonData, err := json.MarshalIndent(tracking, "", "  ")
	if err != nil {
		logger.Error.Printf("Failed to marshal oven tracking data: %v", err)
		return err
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		logger.Error.Printf("Failed to write oven tracking file: %v", err)
		return err
	}

	logger.Info.Printf("Saved oven tracking data: %d cans in oven", len(tracking.Cans))
	return nil
}

// AddCanToOven adds a moisture can to the oven tracking
func AddCanToOven(canNumber, jobNumber, boringNumber, depth, moistureSheet, moistureColumn string) error {
	tracking, err := LoadOvenTracking()
	if err != nil {
		return err
	}

	// Check if can is already in oven
	for _, can := range tracking.Cans {
		if can.CanNumber == canNumber {
			logger.Error.Printf("Can %s is already in the oven (Job: %s, Boring: %s, Depth: %s)",
				canNumber, can.JobNumber, can.BoringNumber, can.Depth)
			return fmt.Errorf("can %s is already in the oven", canNumber)
		}
	}

	newCan := OvenCanData{
		CanNumber:      canNumber,
		JobNumber:      jobNumber,
		BoringNumber:   boringNumber,
		Depth:          depth,
		TimeIn:         time.Now().Format("2006-01-02 15:04:05"),
		MoistureSheet:  moistureSheet,
		MoistureColumn: moistureColumn,
	}

	tracking.Cans = append(tracking.Cans, newCan)

	if err := SaveOvenTracking(tracking); err != nil {
		return err
	}

	logger.Info.Printf("Added can %s to oven (Job: %s, Boring: %s, Depth: %s, Sheet: %s, Column: %s)",
		canNumber, jobNumber, boringNumber, depth, moistureSheet, moistureColumn)
	return nil
}

// RemoveCanFromOven removes a moisture can from the oven tracking
func RemoveCanFromOven(canNumber string) (*OvenCanData, error) {
	tracking, err := LoadOvenTracking()
	if err != nil {
		return nil, err
	}

	var removedCan *OvenCanData
	newCans := []OvenCanData{}

	for _, can := range tracking.Cans {
		if can.CanNumber == canNumber {
			removedCan = &can
		} else {
			newCans = append(newCans, can)
		}
	}

	if removedCan == nil {
		logger.Error.Printf("Can %s is not in the oven", canNumber)
		return nil, fmt.Errorf("can %s is not in the oven", canNumber)
	}

	tracking.Cans = newCans

	if err := SaveOvenTracking(tracking); err != nil {
		return nil, err
	}

	logger.Info.Printf("Removed can %s from oven (Job: %s, Boring: %s, Depth: %s)",
		canNumber, removedCan.JobNumber, removedCan.BoringNumber, removedCan.Depth)
	return removedCan, nil
}

// GetCansInOven returns a list of all cans currently in the oven
func GetCansInOven() ([]OvenCanData, error) {
	tracking, err := LoadOvenTracking()
	if err != nil {
		return nil, err
	}
	return tracking.Cans, nil
}

// IsCanInOven checks if a specific can number is currently in the oven
func IsCanInOven(canNumber string) (bool, *OvenCanData, error) {
	tracking, err := LoadOvenTracking()
	if err != nil {
		return false, nil, err
	}

	for _, can := range tracking.Cans {
		if can.CanNumber == canNumber {
			return true, &can, nil
		}
	}

	return false, nil, nil
}

// GetOvenCanCount returns the number of cans currently in the oven
func GetOvenCanCount() (int, error) {
	tracking, err := LoadOvenTracking()
	if err != nil {
		return 0, err
	}
	return len(tracking.Cans), nil
}

// WriteDryWeightToMoistureSheet writes the dry weight to the moisture sheet for a can
// and calculates: Wt. of water, Dry wt. of soil, and Moisture Content
// Row 11: Can No.
// Row 12: Wet wt. and can
// Row 13: Dry wt. of soil and can (input)
// Row 14: Wt. of water = Row 12 - Row 13
// Row 15: Wt. of can
// Row 16: Dry wt. of soil = Row 13 - Row 15
// Row 17: Moisture Content = (Wt. of water / Dry wt. of soil) * 100
func WriteDryWeightToMoistureSheet(can OvenCanData, dryWeight string) error {
	// Open the Lab file for this job
	filePath := filepath.Join(ProjectRoot, "ex_project", can.JobNumber, fmt.Sprintf("Lab_%s.xlsm", can.JobNumber))

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		logger.Error.Printf("Failed to open Lab file for job %s: %v", can.JobNumber, err)
		return err
	}
	defer f.Close()

	// Read existing values for calculations
	wetWtAndCanCell := fmt.Sprintf("%s12", can.MoistureColumn)
	wtOfCanCell := fmt.Sprintf("%s15", can.MoistureColumn)

	wetWtAndCanStr, _ := f.GetCellValue(can.MoistureSheet, wetWtAndCanCell)
	wtOfCanStr, _ := f.GetCellValue(can.MoistureSheet, wtOfCanCell)

	// Parse values as floats
	var wetWtAndCan, wtOfCan, dryWtAndCan float64
	fmt.Sscanf(wetWtAndCanStr, "%f", &wetWtAndCan)
	fmt.Sscanf(wtOfCanStr, "%f", &wtOfCan)
	fmt.Sscanf(dryWeight, "%f", &dryWtAndCan)

	// Calculate derived values
	wtOfWater := wetWtAndCan - dryWtAndCan       // Row 14
	dryWtOfSoil := dryWtAndCan - wtOfCan         // Row 16
	moistureContent := 0.0
	if dryWtOfSoil > 0 {
		moistureContent = (wtOfWater / dryWtOfSoil) * 100 // Row 17
	}

	// Write all values to the moisture sheet
	f.SetCellValue(can.MoistureSheet, fmt.Sprintf("%s13", can.MoistureColumn), dryWtAndCan)      // Dry wt. of soil and can
	f.SetCellValue(can.MoistureSheet, fmt.Sprintf("%s14", can.MoistureColumn), wtOfWater)        // Wt. of water
	f.SetCellValue(can.MoistureSheet, fmt.Sprintf("%s16", can.MoistureColumn), dryWtOfSoil)      // Dry wt. of soil
	f.SetCellValue(can.MoistureSheet, fmt.Sprintf("%s17", can.MoistureColumn), moistureContent)  // Moisture Content

	// Save the file
	if err := f.Save(); err != nil {
		logger.Error.Printf("Failed to save moisture calculations to Lab file: %v", err)
		return err
	}

	logger.Info.Printf("Wrote moisture calculations to %s column %s (Job: %s, Can: %s):\n"+
		"  Dry wt. of soil and can: %.2f\n"+
		"  Wt. of water: %.2f\n"+
		"  Dry wt. of soil: %.2f\n"+
		"  Moisture Content: %.2f%%",
		can.MoistureSheet, can.MoistureColumn, can.JobNumber, can.CanNumber,
		dryWtAndCan, wtOfWater, dryWtOfSoil, moistureContent)
	return nil
}