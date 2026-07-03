package actions

import (
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
	coreast "github.com/yuin/goldmark/ast"
	gast "github.com/yuin/goldmark/extension/ast"
)

const (
	pdfLineHeight  = 6.0
	pdfBlockMargin = 4.0
)

// renderMarkdownToPDF walks the markdown AST and renders it to the PDF using tr for all text.
func renderMarkdownToPDF(pdf *gofpdf.Fpdf, tr func(string) string, doc coreast.Node, source []byte) {
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		renderBlock(pdf, tr, child, source)
	}
}

func renderBlock(pdf *gofpdf.Fpdf, tr func(string) string, node coreast.Node, source []byte) {
	switch n := node.(type) {
	case *coreast.Document:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			renderBlock(pdf, tr, child, source)
		}
	case *coreast.Heading:
		level := n.Level
		if level > 6 {
			level = 6
		}
		size := float64(22 - level*2)
		if size < 12 {
			size = 12
		}
		pdf.SetFont("Arial", "B", size)
		writeInlineContent(pdf, tr, n, source)
		pdf.Ln(pdfLineHeight + pdfBlockMargin)
		pdf.SetFont("Arial", "", 12)
	case *coreast.Paragraph:
		writeInlineContent(pdf, tr, n, source)
		pdf.Ln(pdfLineHeight + pdfBlockMargin)
	case *coreast.List:
		ordered := n.IsOrdered()
		start := n.Start
		if start <= 0 {
			start = 1
		}
		itemNum := 0
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if item, ok := child.(*coreast.ListItem); ok {
				itemNum++
				var bullet string
				if ordered {
					bullet = tr(fmt.Sprintf("%d. ", start+itemNum-1))
				} else {
					bullet = tr("• ")
				}
				pdf.SetFont("Arial", "", 12)
				pdf.CellFormat(8, pdfLineHeight, bullet, "", 0, "", false, 0, "")
				for inner := item.FirstChild(); inner != nil; inner = inner.NextSibling() {
					renderBlock(pdf, tr, inner, source)
				}
			}
		}
		pdf.Ln(pdfBlockMargin)
	case *coreast.ListItem:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			renderBlock(pdf, tr, child, source)
		}
	case *coreast.CodeBlock:
		pdf.SetFont("Courier", "", 10)
		lit := nodeLiteral(n, source)
		if len(lit) > 0 {
			pdf.MultiCell(0, pdfLineHeight-1, tr(string(lit)), "", "", false)
		}
		pdf.SetFont("Arial", "", 12)
		pdf.Ln(pdfBlockMargin)
	case *coreast.Blockquote:
		left, _, _, _ := pdf.GetMargins()
		saveLeft := left
		pdf.SetLeftMargin(saveLeft + 4)
		pdf.SetX(saveLeft + 4)
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			renderBlock(pdf, tr, child, source)
		}
		pdf.SetLeftMargin(saveLeft)
		pdf.Ln(pdfBlockMargin)
	case *coreast.ThematicBreak:
		pdf.Ln(pdfBlockMargin)
		pdf.Line(pdf.GetX(), pdf.GetY(), pdf.GetX()+190, pdf.GetY())
		pdf.Ln(pdfBlockMargin)
	case *gast.Table:
		renderTable(pdf, tr, n, source)
		pdf.Ln(pdfBlockMargin)
	case *coreast.RawHTML:
		lit := nodeLiteral(n, source)
		if len(lit) > 0 {
			pdf.SetFont("Courier", "", 9)
			pdf.MultiCell(0, pdfLineHeight-1, tr(string(lit)), "", "", false)
			pdf.SetFont("Arial", "", 12)
		}
		pdf.Ln(pdfBlockMargin)
	default:
		if node.FirstChild() != nil {
			writeInlineContent(pdf, tr, node, source)
			pdf.Ln(pdfLineHeight + pdfBlockMargin)
		}
	}
}

const (
	pdfTableLineHt  = 7.0
	pdfTableHeaderR = 72
	pdfTableHeaderG = 72
	pdfTableHeaderB = 72
	pdfTableBorderR = 200
	pdfTableBorderG = 200
	pdfTableBorderB = 200
	pdfTableStripR  = 248
	pdfTableStripG  = 248
	pdfTableStripB  = 248
)

// renderTable draws a markdown table. Table contains TableHeader and TableBody, each with TableRows of TableCells.
func renderTable(pdf *gofpdf.Fpdf, tr func(string) string, table *gast.Table, source []byte) {
	left, _, right, _ := pdf.GetMargins()
	pageW := 210.0
	tblW := pageW - left - right

	// Collect all rows: header rows first, then body (and footer if any)
	var rows [][]string
	var numCols int
	for section := table.FirstChild(); section != nil; section = section.NextSibling() {
		for rowNode := section.FirstChild(); rowNode != nil; rowNode = rowNode.NextSibling() {
			row, ok := rowNode.(*gast.TableRow)
			if !ok {
				continue
			}
			var cells []string
			for c := row.FirstChild(); c != nil; c = c.NextSibling() {
				if cell, ok := c.(*gast.TableCell); ok {
					cells = append(cells, tr(getCellText(cell, source)))
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
				if len(cells) > numCols {
					numCols = len(cells)
				}
			}
		}
	}
	if numCols == 0 {
		return
	}
	colW := tblW / float64(numCols)
	lineHt := pdfTableLineHt

	// Save current colors and set light gray borders for the table
	saveDrawR, saveDrawG, saveDrawB := pdf.GetDrawColor()
	saveFillR, saveFillG, saveFillB := pdf.GetFillColor()
	saveTextR, saveTextG, saveTextB := pdf.GetTextColor()
	pdf.SetDrawColor(pdfTableBorderR, pdfTableBorderG, pdfTableBorderB)

	for i, row := range rows {
		isHeader := i == 0
		lastRow := i == len(rows)-1
		// Header: dark gray background, white text, bold
		if isHeader {
			pdf.SetFont("Arial", "B", 12)
			pdf.SetFillColor(pdfTableHeaderR, pdfTableHeaderG, pdfTableHeaderB)
			pdf.SetTextColor(255, 255, 255)
		} else {
			pdf.SetFont("Arial", "", 12)
			pdf.SetTextColor(0, 0, 0)
			if i%2 == 1 {
				pdf.SetFillColor(pdfTableStripR, pdfTableStripG, pdfTableStripB)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
		}
		border := "LTR"
		if lastRow {
			border = "LTRB"
		}
		fill := true
		for j, cellText := range row {
			w := colW
			if j == numCols-1 {
				w = 0
			}
			pdf.CellFormat(w, lineHt, cellText, border, 0, "L", fill, 0, "")
		}
		pdf.Ln(lineHt)
	}

	// Restore colors and font
	pdf.SetDrawColor(saveDrawR, saveDrawG, saveDrawB)
	pdf.SetFillColor(saveFillR, saveFillG, saveFillB)
	pdf.SetTextColor(saveTextR, saveTextG, saveTextB)
	pdf.SetFont("Arial", "", 12)
}

// getInlineText returns plain text from an inline container (e.g. Image alt text).
func getInlineText(node coreast.Node, source []byte) string {
	var b []byte
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if lit := nodeLiteral(child, source); len(lit) > 0 {
			b = append(b, lit...)
		} else {
			b = append(b, getInlineText(child, source)...)
		}
	}
	return string(b)
}

// getCellText returns plain text from a table cell (walks Paragraph/Text and leaf nodes).
func getCellText(node coreast.Node, source []byte) string {
	var b []byte
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if lit := nodeLiteral(child, source); len(lit) > 0 {
			b = append(b, lit...)
		} else {
			b = append(b, getCellText(child, source)...)
		}
	}
	return string(b)
}

// writeInlineContent outputs inline content (text, strong, emph, code) with correct font changes.
func writeInlineContent(pdf *gofpdf.Fpdf, tr func(string) string, node coreast.Node, source []byte) {
	lineHt := pdfLineHeight
	left, _, right, _ := pdf.GetMargins()
	pageW := 210.0 // A4 mm
	maxW := pageW - left - right

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		writeInline(pdf, tr, child, lineHt, maxW, source)
	}
}

func writeInline(pdf *gofpdf.Fpdf, tr func(string) string, node coreast.Node, lineHt, maxW float64, source []byte) {
	switch n := node.(type) {
	case *coreast.Text:
		lit := n.Segment.Value(source)
		if len(lit) > 0 {
			cellWrap(pdf, tr(string(lit)), lineHt, maxW)
		}
	case *coreast.Emphasis:
		if n.Level >= 2 {
			pdf.SetFont("Arial", "B", 12)
		} else {
			pdf.SetFont("Arial", "I", 12)
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeInline(pdf, tr, c, lineHt, maxW, source)
		}
		pdf.SetFont("Arial", "", 12)
	case *coreast.CodeSpan:
		lit := nodeLiteral(n, source)
		if len(lit) > 0 {
			pdf.SetFont("Courier", "", 11)
			cellWrap(pdf, tr(string(lit)), lineHt, maxW)
			pdf.SetFont("Arial", "", 12)
		}
	case *coreast.Link:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeInline(pdf, tr, c, lineHt, maxW, source)
		}
		if len(n.Destination) > 0 {
			pdf.SetFont("Arial", "I", 10)
			cellWrap(pdf, tr(" ("+string(n.Destination)+")"), lineHt, maxW)
			pdf.SetFont("Arial", "", 12)
		}
	case *coreast.Image:
		alt := getInlineText(n, source)
		if alt != "" {
			cellWrap(pdf, tr(alt), lineHt, maxW)
		}
		if len(n.Destination) > 0 {
			pdf.SetFont("Arial", "I", 10)
			cellWrap(pdf, tr(" [Image: "+string(n.Destination)+"]"), lineHt, maxW)
			pdf.SetFont("Arial", "", 12)
		}
	default:
		if lit := nodeLiteral(node, source); len(lit) > 0 {
			cellWrap(pdf, tr(string(lit)), lineHt, maxW)
		} else if node.FirstChild() != nil {
			for c := node.FirstChild(); c != nil; c = c.NextSibling() {
				writeInline(pdf, tr, c, lineHt, maxW, source)
			}
		}
	}
}

func nodeLiteral(node coreast.Node, source []byte) []byte {
	switch n := node.(type) {
	case *coreast.Text:
		return n.Segment.Value(source)
	case interface{ Text([]byte) []byte }:
		return n.Text(source)
	case interface{ Value([]byte) []byte }:
		return n.Value(source)
	default:
		return nil
	}
}

// cellWrap outputs text with word-wrap: splits on spaces and starts a new line when the next word would overflow.
func cellWrap(pdf *gofpdf.Fpdf, s string, lineHt, maxW float64) {
	left, _, _, _ := pdf.GetMargins()
	words := strings.Fields(s)
	for i, word := range words {
		wordW := pdf.GetStringWidth(word)
		spaceW := 0.0
		if i > 0 {
			spaceW = pdf.GetStringWidth(" ")
		}
		x := pdf.GetX()
		// If this word (and preceding space) would overflow, start a new line first.
		if i > 0 {
			if x+spaceW+wordW > maxW && x > left {
				pdf.Ln(lineHt)
				x = pdf.GetX()
			} else {
				pdf.CellFormat(spaceW, lineHt, " ", "", 0, "", false, 0, "")
				x = pdf.GetX()
			}
		} else if wordW > 0 && x+wordW > maxW && x > left {
			pdf.Ln(lineHt)
			x = pdf.GetX()
		}
		// Single word longer than line width: use MultiCell so it wraps.
		if wordW > maxW-left {
			pdf.MultiCell(0, lineHt, word, "", "", false)
		} else {
			if x+wordW > maxW && x > left {
				pdf.Ln(lineHt)
			}
			pdf.CellFormat(wordW, lineHt, word, "", 0, "", false, 0, "")
		}
	}
}
