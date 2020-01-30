package ui

import (
	"fmt"
	"image"

	. "github.com/gizak/termui/v3"
)

type Table struct {
	*Block
	Header []string
	Rows   [][]string
	Styles [][]*Style

	ColWidths []int
	ColGap    int
	PadLeft   int

	ShowLocation bool

	SelectedRow int
	TopRow      int

	ColResizer func()
}

func NewTable() *Table {
	return &Table{
		Block:       NewBlock(),
		SelectedRow: 0,
		TopRow:      0,
		ColResizer:  func() {},
	}
}

func (table *Table) Draw(buf *Buffer) {
	table.Block.Draw(buf)

	if table.ShowLocation {
		table.drawLocation(buf)
	}

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

func (self *Table) calcPos() {
	if self.SelectedRow < 0 {
		self.SelectedRow = 0
	}
	if self.SelectedRow < self.TopRow {
		self.TopRow = self.SelectedRow
	}

	if self.SelectedRow > len(self.Rows)-1 {
		self.SelectedRow = len(self.Rows) - 1
	}
	if self.SelectedRow > self.TopRow+(self.Inner.Dy()-2) {
		self.TopRow = self.SelectedRow - (self.Inner.Dy() - 2)
	}
}

func (self *Table) ScrollUp() {
	self.SelectedRow--
	self.calcPos()
}

func (self *Table) ScrollDown() {
	self.SelectedRow++
	self.calcPos()
}
