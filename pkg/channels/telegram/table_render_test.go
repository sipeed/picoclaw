package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuneWidth(t *testing.T) {
	assert.Equal(t, 1, runeWidth('a'))
	assert.Equal(t, 1, runeWidth('1'))
	assert.Equal(t, 2, runeWidth('名'))
	assert.Equal(t, 2, runeWidth('ア'))
	assert.Equal(t, 2, runeWidth('あ'))
}

func TestStringWidth(t *testing.T) {
	assert.Equal(t, 5, stringWidth("hello"))
	assert.Equal(t, 4, stringWidth("名前"))   // 2+2
	assert.Equal(t, 4, stringWidth("Go言"))   // 1+1+2
}

func TestRenderTableMono(t *testing.T) {
	td := tableData{
		headers: []string{"Name", "Version", "Status"},
		rows: [][]string{
			{"Go", "1.22", "Active"},
			{"Python", "3.12", "Active"},
		},
	}

	got := renderTableMono(td)
	expected := "Name   | Version | Status\n" +
		"-------+---------+-------\n" +
		"Go     | 1.22    | Active\n" +
		"Python | 3.12    | Active"
	assert.Equal(t, expected, got)
}

func TestRenderTableMono_SingleColumn(t *testing.T) {
	td := tableData{
		headers: []string{"Item"},
		rows: [][]string{
			{"Apple"},
			{"Banana"},
		},
	}

	got := renderTableMono(td)
	expected := "Item  \n" +
		"------\n" +
		"Apple \n" +
		"Banana"
	assert.Equal(t, expected, got)
}

func TestRenderTableMono_CJK(t *testing.T) {
	td := tableData{
		headers: []string{"名前", "バージョン"},
		rows: [][]string{
			{"Go", "1.22"},
		},
	}

	got := renderTableMono(td)
	// "名前" width=4, "バージョン" width=10
	// "Go" width=2, "1.22" width=4
	expected := "名前 | バージョン\n" +
		"-----+-----------\n" +
		"Go   | 1.22      "
	assert.Equal(t, expected, got)
}

func TestRenderTableMono_EmptyCells(t *testing.T) {
	td := tableData{
		headers: []string{"A", "B"},
		rows: [][]string{
			{"", "2"},
			{"1", ""},
		},
	}

	got := renderTableMono(td)
	expected := "A | B\n" +
		"--+--\n" +
		"  | 2\n" +
		"1 |  "
	assert.Equal(t, expected, got)
}

func TestRenderTableAsListHTML(t *testing.T) {
	td := tableData{
		headers: []string{"Name", "Version"},
		rows: [][]string{
			{"Go", "1.22"},
			{"Python", "3.12"},
		},
	}

	got := renderTableAsListHTML(td)
	expected := "<b>Row 1:</b>\n• Name: Go\n• Version: 1.22\n\n" +
		"<b>Row 2:</b>\n• Name: Python\n• Version: 3.12"
	assert.Equal(t, expected, got)
}

func TestRenderTableAsListHTML_Escaping(t *testing.T) {
	td := tableData{
		headers: []string{"A&B"},
		rows: [][]string{
			{"<val>"},
		},
	}

	got := renderTableAsListHTML(td)
	assert.Contains(t, got, "A&amp;B")
	assert.Contains(t, got, "&lt;val&gt;")
}

func TestRenderTableAsListMDV2(t *testing.T) {
	td := tableData{
		headers: []string{"Name", "Version"},
		rows: [][]string{
			{"Go", "1.22"},
		},
	}

	got := renderTableAsListMDV2(td)
	expected := "*Row 1:*\n• Name: Go\n• Version: 1\\.22"
	assert.Equal(t, expected, got)
}

func TestTableWidth(t *testing.T) {
	td := tableData{
		headers: []string{"A", "BB", "CCC"},
		rows: [][]string{
			{"x", "yy", "zzz"},
		},
	}
	// col widths: 1, 2, 3; separators: 3*2=6; total=12
	assert.Equal(t, 12, tableWidth(td))
}

func TestTableWidth_CJK(t *testing.T) {
	td := tableData{
		headers: []string{"名前", "値"},
		rows:    [][]string{{"Go", "ok"}},
	}
	// "名前"=4, "値"=2 → max col widths: 4, 2 → total=4+2+3=9
	assert.Equal(t, 9, tableWidth(td))
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "1", itoa(1))
	assert.Equal(t, "9", itoa(9))
	assert.Equal(t, "10", itoa(10))
	assert.Equal(t, "42", itoa(42))
	assert.Equal(t, "100", itoa(100))
}
