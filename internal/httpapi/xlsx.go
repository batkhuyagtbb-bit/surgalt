package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"surgalt/internal/store"
)

// Тестийн Excel загвар: нэг мөр = нэг асуулт.
var quizSheetHeader = []string{"Төрөл", "Асуулт", "Хариулт 1", "Хариулт 2", "Хариулт 3", "Хариулт 4", "Хариулт 5", "Хариулт 6", "Зөв хариулт", "Оноо", "Тайлбар"}

var quizSheetExamples = [][]string{
	{"нэг", "Монгол улсын нийслэл аль вэ?", "Дархан", "Улаанбаатар", "Эрдэнэт", "", "", "", "2", "1", "Улаанбаатар 1639 оноос"},
	{"олон", "Аль нь анхны тоо вэ?", "2", "4", "5", "9", "", "", "1,3", "2", ""},
	{"бичих", "5 × 6 = ?", "", "", "", "", "", "", "30; гучин", "1", ""},
	{"харгалзуулах", "Улсыг нийслэлтэй нь холбоно уу", "Япон=Токио", "Франц=Парис", "Хятад=Бээжин", "", "", "", "", "3", "Хариулт баганад «зүүн=баруун» гэж бичнэ"},
}

// handleQuizTemplate: GET /api/quiz-template.xlsx — багшид татаж бөглөх загвар.
func (s *Server) handleQuizTemplate(w http.ResponseWriter, r *http.Request) {
	rows := append([][]string{quizSheetHeader}, quizSheetExamples...)
	rows = append(rows, []string{}, []string{"Төрөл: нэг (нэг сонголт), олон (олон сонголт), бичих (бичгээр), харгалзуулах (хос). Зөв хариулт: дугаараар (1,3), бичих төрөлд «;»-ээр тусгаарлана."})
	b, err := writeXLSX("Тест", rows)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "загвар үүсгэж чадсангүй")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="surgalt-test-zagvar.xlsx"`)
	_, _ = w.Write(b)
}

// handleQuizImport: POST /api/me/quiz-import (multipart "file": .xlsx эсвэл .csv) → асуулт блокууд.
// Хадгалахгүй: багш засварлагч дээр шалгаад хадгална.
func (s *Server) handleQuizImport(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Excel файлаа сонгоно уу (5MB хүртэл)")
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "файлыг уншиж чадсангүй")
		return
	}
	var rows [][]string
	name := strings.ToLower(hdr.Filename)
	switch {
	case strings.HasSuffix(name, ".xlsx"):
		rows, err = readXLSX(raw)
	case strings.HasSuffix(name, ".csv"):
		cr := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})))
		cr.FieldsPerRecord, cr.LazyQuotes = -1, true
		rows, err = cr.ReadAll()
	default:
		writeErr(w, http.StatusBadRequest, "зөвхөн .xlsx эсвэл .csv файл")
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, "файлыг уншиж чадсангүй: "+err.Error())
		return
	}
	blocks, problems := parseQuizRows(rows, c.UID)
	if len(blocks) == 0 && len(problems) == 0 {
		problems = append(problems, "асуулт олдсонгүй — загварын дагуу бөглөнө үү")
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": blocks, "problems": problems})
}

func parseQuizRows(rows [][]string, teacherID string) ([]store.Block, []string) {
	kinds := map[string]string{"нэг": "single", "нэг сонголт": "single", "single": "single", "олон": "multi", "олон сонголт": "multi", "multi": "multi",
		"бичих": "text", "бичгээр": "text", "text": "text", "харгалзуулах": "match", "хос": "match", "match": "match"}
	cell := func(row []string, i int) string {
		if i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}
	out := []store.Block{}
	problems := []string{}
	for n, row := range rows {
		kind := kinds[strings.ToLower(cell(row, 0))]
		if n == 0 || kind == "" { // гарчиг, хоосон, тайлбар мөр
			if n > 0 && cell(row, 0) != "" && cell(row, 1) != "" {
				problems = append(problems, fmt.Sprintf("%d-р мөр: төрөл «%s» танигдсангүй", n+1, cell(row, 0)))
			}
			continue
		}
		if len(out) >= maxBlocks {
			problems = append(problems, "хэт олон асуулт — эхний 120-г авлаа")
			break
		}
		q := &store.Quiz{Kind: kind, Question: cell(row, 1), Explain: cell(row, 10)}
		q.Points, _ = strconv.Atoi(cell(row, 9))
		var opts []string
		for i := 2; i <= 7; i++ {
			if v := cell(row, i); v != "" {
				opts = append(opts, v)
			}
		}
		ans := cell(row, 8)
		switch kind {
		case "single", "multi":
			q.Options = opts
			for _, p := range strings.FieldsFunc(ans, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
				if k, err := strconv.Atoi(p); err == nil {
					q.Correct = append(q.Correct, k-1)
				}
			}
		case "text":
			for _, p := range strings.Split(ans, ";") {
				if p = strings.TrimSpace(p); p != "" {
					q.Answers = append(q.Answers, p)
				}
			}
		case "match":
			for _, o := range opts {
				l, rr, ok := strings.Cut(o, "=")
				if !ok {
					q.Left = nil
					break
				}
				q.Left, q.Right = append(q.Left, strings.TrimSpace(l)), append(q.Right, strings.TrimSpace(rr))
			}
		}
		if msg := validateQuiz(q, teacherID); msg != "" {
			problems = append(problems, fmt.Sprintf("%d-р мөр: %s", n+1, msg))
			continue
		}
		out = append(out, store.Block{ID: fmt.Sprintf("imp%05d", len(out)+1), Type: "quiz", Quiz: q})
	}
	return out, problems
}

// ---- хамгийн бага XLSX уншигч/бичигч (гадны сангүй) ----

func readXLSX(raw []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("xlsx биш байна")
	}
	open := func(name string) ([]byte, error) {
		for _, f := range zr.File {
			if f.Name == name {
				if f.UncompressedSize64 > 50<<20 {
					return nil, fmt.Errorf("хэт том")
				}
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(io.LimitReader(rc, 50<<20))
			}
		}
		return nil, nil
	}
	var shared []string
	if b, err := open("xl/sharedStrings.xml"); err != nil {
		return nil, err
	} else if b != nil {
		var sst struct {
			SI []struct {
				T string `xml:"t"`
				R []struct {
					T string `xml:"t"`
				} `xml:"r"`
			} `xml:"si"`
		}
		if err := xml.Unmarshal(b, &sst); err != nil {
			return nil, err
		}
		for _, si := range sst.SI {
			t := si.T
			for _, r := range si.R {
				t += r.T
			}
			shared = append(shared, t)
		}
	}
	sheet, err := open("xl/worksheets/sheet1.xml")
	if err != nil || sheet == nil {
		return nil, fmt.Errorf("эхний хуудас олдсонгүй")
	}
	var ws struct {
		Rows []struct {
			C []struct {
				R  string `xml:"r,attr"`
				T  string `xml:"t,attr"`
				V  string `xml:"v"`
				IS struct {
					T string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(sheet, &ws); err != nil {
		return nil, err
	}
	var rows [][]string
	for _, row := range ws.Rows {
		var cells []string
		for _, c := range row.C {
			col := 0
			for _, ch := range c.R {
				if ch < 'A' || ch > 'Z' {
					break
				}
				col = col*26 + int(ch-'A'+1)
			}
			col--
			if col < 0 || col > 50 {
				continue
			}
			v := c.V
			switch c.T {
			case "s":
				if i, err := strconv.Atoi(c.V); err == nil && i >= 0 && i < len(shared) {
					v = shared[i]
				}
			case "inlineStr":
				v = c.IS.T
			}
			for len(cells) <= col {
				cells = append(cells, "")
			}
			cells[col] = v
		}
		rows = append(rows, cells)
		if len(rows) > 2000 {
			break
		}
	}
	return rows, nil
}

func writeXLSX(sheetName string, rows [][]string) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><cols><col min="1" max="1" width="14" customWidth="1"/><col min="2" max="2" width="44" customWidth="1"/><col min="3" max="8" width="18" customWidth="1"/><col min="9" max="9" width="16" customWidth="1"/><col min="10" max="10" width="8" customWidth="1"/><col min="11" max="11" width="36" customWidth="1"/></cols><sheetData>`)
	for i, row := range rows {
		fmt.Fprintf(&sb, `<row r="%d">`, i+1)
		for j, v := range row {
			if v == "" {
				continue
			}
			style := ""
			if i == 0 {
				style = ` s="1"`
			}
			var esc bytes.Buffer
			_ = xml.EscapeText(&esc, []byte(v))
			fmt.Fprintf(&sb, `<c r="%c%d" t="inlineStr"%s><is><t>%s</t></is></c>`, 'A'+j, i+1, style, esc.String())
		}
		sb.WriteString(`</row>`)
	}
	sb.WriteString(`</sheetData></worksheet>`)
	var sheetEsc bytes.Buffer
	_ = xml.EscapeText(&sheetEsc, []byte(sheetName))
	files := []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="` + sheetEsc.String() + `" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`},
		{"xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="2"><xf/><xf fontId="1" applyFont="1"/></cellXfs></styleSheet>`},
		{"xl/worksheets/sheet1.xml", sb.String()},
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		wr, err := zw.Create(f.name)
		if err != nil {
			return nil, err
		}
		if _, err := wr.Write([]byte(f.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
