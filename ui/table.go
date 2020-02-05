package ui

import (
	"fmt"
	"image"

	. "github.com/gizak/termui/v3"
)

// Table is custom table implemented on top of the basic Block
// It allows for an arbitrary number of rows and columns with no separators
// and per entry styling.
type Table struct {
	*Block
	Header []string
	Rows   [][]string
	Styles [][]*Style

	Footer string

	ColWidths []int
	ColGap    int
	PadLeft   int

	SelectedRow int
	TopRow      int

	ColResizer func()
}

// NewTable constructs a new table
func NewTable() *Table {
	return &Table{
		Block:       NewBlock(),
		SelectedRow: 0,
		TopRow:      0,
		ColResizer:  func() {},
	}
}

// Draw draws the table to the given buffer
func (table *Table) Draw(buf *Buffer) {
	table.Block.Draw(buf)

	table.drawLocation(buf)
	table.drawUpdated(buf)
	table.ColResizer()

	colXPos := []int{}
	cur := 1 + table.PadLeft
	for _, w := range table.ColWidths {
		colXPos = append(colXPos, cur)
		cur += w
		cur += table.ColGap
	}

	for i, h := range table.Header {
		width := table.ColWidths[i]
		if width == 0 {
			continue
		}

		if width > (table.Inner.Dx()-colXPos[i])+1 {
			continue
		}
		buf.SetString(
			h,
			NewStyle(Theme.Default.Fg, ColorClear, ModifierBold),
			image.Pt(table.Inner.Min.X+colXPos[i]-1, table.Inner.Min.Y),
		)
	}

	if table.TopRow < 0 {
		return
	}

	for rowNum := table.TopRow; rowNum < table.TopRow+table.Inner.Dy()-1 && rowNum < len(table.Rows); rowNum++ {
		row := table.Rows[rowNum]
		y := (rowNum + 2) - table.TopRow

		style := NewStyle(Theme.Default.Fg)
		for i, width := range table.ColWidths {
			if width == 0 {
				continue
			}
			if width > (table.Inner.Dx()-colXPos[i])+1 {
				continue
			}
			r := TrimString(row[i], width)
			if table.Styles[rowNum][i] != nil {
				buf.SetString(
					r,
					*table.Styles[rowNum][i],
					image.Pt(table.Inner.Min.X+colXPos[i]-1, table.Inner.Min.Y+y-1),
				)
			} else {
				buf.SetString(
					r,
					style,
					image.Pt(table.Inner.Min.X+colXPos[i]-1, table.Inner.Min.Y+y-1),
				)
			}
		}
	}
}

func (table *Table) drawLocation(buf *Buffer) {
	total := len(table.Rows)
	topRow := table.TopRow + 1
	bottomRow := table.TopRow + table.Inner.Dy() - 1
	if bottomRow > total {
		bottomRow = total
	}

	loc := fmt.Sprintf(" %d - %d of %d ", topRow, bottomRow, total)

	width := len(loc)
	buf.SetString(loc, table.TitleStyle, image.Pt(table.Max.X-width-2, table.Min.Y))
}

func (table *Table) drawUpdated(buf *Buffer) {
	width := len(table.Footer)
	buf.SetString(table.Footer, table.TitleStyle, image.Pt(table.Max.X/2-width/2, table.Max.Y-1))
}

func (table *Table) calcPos() {
	if table.SelectedRow < 0 {
		table.SelectedRow = 0
	}
	if table.SelectedRow < table.TopRow {
		table.TopRow = table.SelectedRow
	}

	if table.SelectedRow > len(table.Rows)-1 {
		table.SelectedRow = len(table.Rows) - 1
	}
	if table.SelectedRow > table.TopRow+(table.Inner.Dy()-2) {
		table.TopRow = table.SelectedRow - (table.Inner.Dy() - 2)
	}
}

// -------------------------------------------------------------------------

// ScrollUp scrolls up the cursor in the table
func (table *Table) ScrollUp() {
	table.SelectedRow--
	table.calcPos()
}

// ScrollDown scrolls down the cursor in the table
func (table *Table) ScrollDown() {
	table.SelectedRow++
	table.calcPos()
}
