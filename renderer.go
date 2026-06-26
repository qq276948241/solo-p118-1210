package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Renderer struct {
	Wall         tcell.Style
	Floor        tcell.Style
	Obstacle     tcell.Style
	RedFood      tcell.Style
	GoldFood     tcell.Style
	PowerUp      tcell.Style
	SnakeBody    tcell.Style
	SnakeHead    tcell.Style
	WallPassHead tcell.Style
	Score        tcell.Style
	Buff         tcell.Style
	Help         tcell.Style

	OverlayStart  tcell.Style
	OverlayPaused tcell.Style
	OverlayOver   tcell.Style
}

func NewRenderer() *Renderer {
	return &Renderer{
		Wall:         tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorGray),
		Floor:        tcell.StyleDefault.Background(tcell.ColorBlack),
		Obstacle:     tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkGray),
		RedFood:      tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack),
		GoldFood:     tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack),
		PowerUp:      tcell.StyleDefault.Foreground(tcell.ColorDarkCyan).Background(tcell.ColorBlack),
		SnakeBody:    tcell.StyleDefault.Foreground(tcell.ColorLime).Background(tcell.ColorBlack),
		SnakeHead:    tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack).Bold(true),
		WallPassHead: tcell.StyleDefault.Foreground(tcell.ColorDarkCyan).Background(tcell.ColorBlack).Bold(true),
		Score:        tcell.StyleDefault.Foreground(tcell.ColorWhite),
		Buff:         tcell.StyleDefault.Foreground(tcell.ColorDarkCyan),
		Help:         tcell.StyleDefault.Foreground(tcell.ColorDimGray),

		OverlayStart:  tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorNavy),
		OverlayPaused: tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue),
		OverlayOver:   tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkRed),
	}
}

func (r *Renderer) comboColor(mult int) tcell.Color {
	switch mult {
	case 1:
		return tcell.ColorWhite
	case 2:
		return tcell.ColorYellow
	case 3:
		return tcell.ColorOrange
	default:
		return tcell.ColorRed
	}
}

func (r *Renderer) comboStyle(mult int) tcell.Style {
	if mult >= 5 {
		return tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true).Blink(true)
	}
	if mult >= 4 {
		return tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	}
	return tcell.StyleDefault.Foreground(r.comboColor(mult)).Bold(true)
}

func (r *Renderer) comboHeadStyle(mult int) tcell.Style {
	if mult >= 5 {
		return tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true).Blink(true)
	}
	if mult >= 4 {
		return tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
	}
	return tcell.StyleDefault.Foreground(r.comboColor(mult)).Background(tcell.ColorBlack).Bold(true)
}

func (r *Renderer) setCell(s tcell.Screen, x, y int, ch rune, style tcell.Style) {
	s.SetContent(x, y, ch, nil, style)
}

func (r *Renderer) drawText(s tcell.Screen, x, y int, text string, style tcell.Style) {
	for i, ch := range text {
		r.setCell(s, x+i, y, ch, style)
	}
}

func (r *Renderer) overlay(s tcell.Screen, ox, oy, w, h int, text string, style tcell.Style) {
	lines := strings.Split(text, "\n")
	maxW := 0
	for _, l := range lines {
		if len(l) > maxW {
			maxW = len(l)
		}
	}
	boxW := maxW + 4
	boxH := len(lines) + 2
	bx := ox + (w-boxW)/2
	by := oy + (h-boxH)/2

	for dy := 0; dy < boxH; dy++ {
		for dx := 0; dx < boxW; dx++ {
			r.setCell(s, bx+dx, by+dy, ' ', style)
		}
	}

	borderStyle := style.Bold(true)
	for dx := 0; dx < boxW; dx++ {
		r.setCell(s, bx+dx, by, '─', borderStyle)
		r.setCell(s, bx+dx, by+boxH-1, '─', borderStyle)
	}
	for dy := 0; dy < boxH; dy++ {
		r.setCell(s, bx, by+dy, '│', borderStyle)
		r.setCell(s, bx+boxW-1, by+dy, '│', borderStyle)
	}
	r.setCell(s, bx, by, '┌', borderStyle)
	r.setCell(s, bx+boxW-1, by, '┐', borderStyle)
	r.setCell(s, bx, by+boxH-1, '└', borderStyle)
	r.setCell(s, bx+boxW-1, by+boxH-1, '┘', borderStyle)

	for i, l := range lines {
		r.drawText(s, bx+2, by+1+i, l, style)
	}
}

func (r *Renderer) powerUpGlyph(kind PowerType) rune {
	switch kind {
	case PowerSlow:
		return '◇'
	case PowerWallPass:
		return '◈'
	case PowerShrink:
		return '▽'
	default:
		return '?'
	}
}

func (r *Renderer) Draw(s tcell.Screen, g *Game) {
	s.Clear()

	ox, oy := 1, 1

	for x := 0; x < MapW; x++ {
		r.setCell(s, ox+x, oy, ' ', r.Wall)
		r.setCell(s, ox+x, oy+MapH-1, ' ', r.Wall)
	}
	for y := 1; y < MapH-1; y++ {
		r.setCell(s, ox, oy+y, ' ', r.Wall)
		r.setCell(s, ox+MapW-1, oy+y, ' ', r.Wall)
	}

	for y := 1; y < MapH-1; y++ {
		for x := 1; x < MapW-1; x++ {
			r.setCell(s, ox+x, oy+y, ' ', r.Floor)
		}
	}

	for _, o := range g.obstacles {
		r.setCell(s, ox+o.X, oy+o.Y, '█', r.Obstacle)
	}

	for _, f := range g.foods {
		r.setCell(s, ox+f.X, oy+f.Y, '●', r.RedFood)
	}

	r.setCell(s, ox+g.goldFood.X, oy+g.goldFood.Y, '★', r.GoldFood)

	if g.powerUp != nil {
		r.setCell(s, ox+g.powerUp.Pos.X, oy+g.powerUp.Pos.Y, r.powerUpGlyph(g.powerUp.Kind), r.PowerUp)
	}

	for i, seg := range g.snake {
		if i == len(g.snake)-1 {
			hStyle := r.SnakeHead
			if now := time.Now(); !g.comboFlashEnd.IsZero() && now.Before(g.comboFlashEnd) {
				hStyle = r.comboHeadStyle(g.currentCombo)
			}
			r.setCell(s, ox+seg.X, oy+seg.Y, '◉', hStyle)
		} else {
			r.setCell(s, ox+seg.X, oy+seg.Y, '■', r.SnakeBody)
		}
	}

	if g.wallPass {
		head := g.snake[len(g.snake)-1]
		r.setCell(s, ox+head.X, oy+head.Y, '◉', r.WallPassHead)
	}

	infoY := oy + MapH + 1
	scoreText := fmt.Sprintf("Score: %d   High: %d   Length: %d", g.score, g.highScore, len(g.snake))
	showCombo := g.currentCombo > 0
	if !g.lastEatTime.IsZero() && time.Since(g.lastEatTime) >= 2*time.Second {
		showCombo = false
	}
	r.setCell(s, ox, infoY, ' ', r.Score)
	r.drawText(s, ox, infoY, scoreText, r.Score)
	if showCombo {
		comboStr := fmt.Sprintf("  x%d", g.currentCombo)
		r.drawText(s, ox+len(scoreText), infoY, comboStr, r.comboStyle(g.currentCombo))
	}

	statusY := infoY + 1
	now := time.Now()
	status := ""
	if g.speedBoost && now.Before(g.boostEnd) {
		remain := g.boostEnd.Sub(now).Truncate(time.Second / 2)
		status += fmt.Sprintf("  SPEED x2 %.0fs", remain.Seconds())
	}
	if !g.slowEnd.IsZero() && now.Before(g.slowEnd) {
		remain := g.slowEnd.Sub(now).Truncate(time.Second / 2)
		status += fmt.Sprintf("  SLOW %.0fs", remain.Seconds())
	}
	if g.wallPass {
		status += "  WALL-PASS"
	}
	if status != "" {
		r.drawText(s, ox, statusY, status, r.Buff)
	}

	helpY := statusY + 1
	r.drawText(s, ox, helpY, "Arrow:Move  P:Pause  Q:Quit", r.Help)

	if g.startup {
		r.overlay(s, ox, oy, MapW, MapH,
			fmt.Sprintf("SNAKE GAME\n\nHistorical Best: %d\n\nPress any key to start", g.highScore),
			r.OverlayStart)
	}

	if g.paused && !g.over {
		r.overlay(s, ox, oy, MapW, MapH,
			"PAUSED\n\nPress P to resume",
			r.OverlayPaused)
	}

	if g.over {
		r.overlay(s, ox, oy, MapW, MapH,
			fmt.Sprintf("GAME OVER\n\nScore: %d\nHigh Score: %d\nSnake Length: %d\n\nPress R to restart", g.score, g.highScore, len(g.snake)),
			r.OverlayOver)
	}
}
