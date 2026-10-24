package parsers

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/diego/go-refinery/internal/ports"
)

const (
	maxXLSXSharedStringsBytes = 64 * 1024 * 1024  // 64 MB cap on sharedStrings.xml
	maxXLSXWorksheetBytes     = 512 * 1024 * 1024 // 512 MB cap on sheet XML stream (Zip Bomb guard)
)

type xlsxIterator struct {
	rawCloser     io.Closer
	tempFile      *os.File
	zipReader     *zip.ReadCloser
	sheetRC       io.ReadCloser
	decoder       *xml.Decoder
	sharedStrings []string
	headers       []string
}

type xlsxCell struct {
	Ref        string `xml:"r,attr"`
	Type       string `xml:"t,attr"`
	Value      string `xml:"v"`
	InlineText string `xml:"is>t"`
}

type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}

// newXLSXIterator streams rows from an OpenXML (.xlsx) workbook using standard library
// archive/zip and encoding/xml.Decoder without loading the entire worksheet into memory.
func newXLSXIterator(rc io.ReadCloser) (ports.RowIterator, error) {
	var (
		zipPath  string
		tempFile *os.File
	)

	if f, ok := rc.(*os.File); ok {
		zipPath = f.Name()
	} else {
		tf, err := os.CreateTemp("", "refinery-xlsx-*.xlsx")
		if err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("creating temporary xlsx file: %w", err)
		}
		tempFile = tf
		zipPath = tf.Name()
		if _, err := io.Copy(tf, io.LimitReader(rc, maxXLSXWorksheetBytes)); err != nil {
			_ = tf.Close()
			_ = os.Remove(zipPath)
			_ = rc.Close()
			return nil, fmt.Errorf("spilling xlsx stream to disk: %w", err)
		}
	}

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		cleanupXLSXTemp(rc, tempFile)
		return nil, fmt.Errorf("opening xlsx zip archive: %w", err)
	}

	sharedStrings, err := loadSharedStrings(&zr.Reader)
	if err != nil {
		_ = zr.Close()
		cleanupXLSXTemp(rc, tempFile)
		return nil, err
	}

	sheetFile, err := findFirstWorksheet(&zr.Reader)
	if err != nil {
		_ = zr.Close()
		cleanupXLSXTemp(rc, tempFile)
		return nil, err
	}

	sheetRC, err := sheetFile.Open()
	if err != nil {
		_ = zr.Close()
		cleanupXLSXTemp(rc, tempFile)
		return nil, fmt.Errorf("opening worksheet xml stream: %w", err)
	}

	limitedSheet := &limitedReadCloser{
		Reader: io.LimitReader(sheetRC, maxXLSXWorksheetBytes),
		Closer: sheetRC,
	}

	it := &xlsxIterator{
		rawCloser:     rc,
		tempFile:      tempFile,
		zipReader:     zr,
		sheetRC:       limitedSheet,
		decoder:       xml.NewDecoder(limitedSheet),
		sharedStrings: sharedStrings,
	}

	headers, err := it.readNextRow()
	if err != nil && !errors.Is(err, io.EOF) {
		_ = it.Close()
		return nil, fmt.Errorf("reading xlsx header row: %w", err)
	}
	it.headers = headers

	return it, nil
}

func (it *xlsxIterator) Headers() []string {
	return it.headers
}

func (it *xlsxIterator) Next() ([]string, error) {
	return it.readNextRow()
}

func (it *xlsxIterator) Close() error {
	if it.sheetRC != nil {
		_ = it.sheetRC.Close()
	}
	if it.zipReader != nil {
		_ = it.zipReader.Close()
	}
	cleanupXLSXTemp(it.rawCloser, it.tempFile)
	return nil
}

func (it *xlsxIterator) readNextRow() ([]string, error) {
	for {
		tok, err := it.decoder.Token()
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "row" {
			continue
		}

		var r xlsxRow
		if err := it.decoder.DecodeElement(&r, &se); err != nil {
			return nil, fmt.Errorf("decoding xlsx <row>: %w", err)
		}

		return it.extractCells(r.Cells), nil
	}
}

func (it *xlsxIterator) extractCells(cells []xlsxCell) []string {
	if len(cells) == 0 {
		return []string{}
	}

	maxCol := -1
	indexed := make(map[int]string, len(cells))

	for i, c := range cells {
		colIdx := parseCellColumnIndex(c.Ref)
		if colIdx < 0 {
			colIdx = i
		}
		if colIdx > maxCol {
			maxCol = colIdx
		}
		indexed[colIdx] = it.resolveCellValue(c)
	}

	if len(it.headers) > 0 && len(it.headers)-1 > maxCol {
		maxCol = len(it.headers) - 1
	}

	row := make([]string, maxCol+1)
	for idx, val := range indexed {
		if idx >= 0 && idx < len(row) {
			row[idx] = val
		}
	}
	return row
}

func (it *xlsxIterator) resolveCellValue(c xlsxCell) string {
	switch c.Type {
	case "s":
		var idx int
		if _, err := fmt.Sscanf(strings.TrimSpace(c.Value), "%d", &idx); err == nil {
			if idx >= 0 && idx < len(it.sharedStrings) {
				return it.sharedStrings[idx]
			}
		}
		return c.Value
	case "inlineStr":
		return c.InlineText
	case "b":
		if strings.TrimSpace(c.Value) == "1" {
			return "true"
		}
		return "false"
	default:
		return c.Value
	}
}

func parseCellColumnIndex(cellRef string) int {
	if cellRef == "" {
		return -1
	}
	col := 0
	hasLetters := false
	for i := 0; i < len(cellRef); i++ {
		ch := cellRef[i]
		if ch >= 'A' && ch <= 'Z' {
			hasLetters = true
			col = col*26 + int(ch-'A'+1)
		} else if ch >= 'a' && ch <= 'z' {
			hasLetters = true
			col = col*26 + int(ch-'a'+1)
		} else {
			break
		}
	}
	if !hasLetters {
		return -1
	}
	return col - 1
}

func loadSharedStrings(zr *zip.Reader) ([]string, error) {
	var ssFile *zip.File
	for _, f := range zr.File {
		if strings.EqualFold(f.Name, "xl/sharedStrings.xml") {
			ssFile = f
			break
		}
	}
	if ssFile == nil {
		return nil, nil
	}

	rc, err := ssFile.Open()
	if err != nil {
		return nil, fmt.Errorf("opening xl/sharedStrings.xml: %w", err)
	}
	defer rc.Close()

	dec := xml.NewDecoder(io.LimitReader(rc, maxXLSXSharedStringsBytes))
	var result []string

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parsing sharedStrings.xml: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "si" {
			continue
		}

		var si struct {
			Text string   `xml:"t"`
			Runs []string `xml:"r>t"`
		}
		if err := dec.DecodeElement(&si, &se); err != nil {
			return nil, fmt.Errorf("decoding <si> element: %w", err)
		}
		if len(si.Runs) > 0 {
			result = append(result, strings.Join(si.Runs, ""))
		} else {
			result = append(result, si.Text)
		}
	}
	return result, nil
}

func findFirstWorksheet(zr *zip.Reader) (*zip.File, error) {
	var sheets []*zip.File
	for _, f := range zr.File {
		lower := strings.ToLower(f.Name)
		if strings.HasPrefix(lower, "xl/worksheets/sheet") && strings.HasSuffix(lower, ".xml") {
			sheets = append(sheets, f)
		}
	}
	if len(sheets) == 0 {
		return nil, errors.New("xlsx archive contains no worksheets under xl/worksheets/")
	}
	sort.Slice(sheets, func(i, j int) bool {
		return sheets[i].Name < sheets[j].Name
	})
	return sheets[0], nil
}

func cleanupXLSXTemp(rawCloser io.Closer, tempFile *os.File) {
	if rawCloser != nil {
		_ = rawCloser.Close()
	}
	if tempFile != nil {
		name := tempFile.Name()
		_ = tempFile.Close()
		_ = os.Remove(name)
	}
}

type limitedReadCloser struct {
	io.Reader
	io.Closer
}
