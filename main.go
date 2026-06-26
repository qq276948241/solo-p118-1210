package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	MapW        = 20
	MapH        = 20
	NumObstacle = 8
	NumRedFood  = 3
	NumGoldFood = 1
)

type Point struct {
	X, Y int
}

type PowerType int

const (
	PowerSlow PowerType = iota
	PowerWallPass
	PowerShrink
)

type PowerUp struct {
	Pos       Point
	Kind      PowerType
	SpawnTime time.Time
	Duration  time.Duration
}

type Game struct {
	snake      []Point
	dir        Point
	nextDir    Point
	foods      []Point
	goldFood   Point
	obstacles  []Point
	powerUp    *PowerUp
	powerIndex int

	score      int
	highScore  int
	baseSpeed  time.Duration
	speedBoost bool
	boostEnd   time.Time
	slowEnd    time.Time
	wallPass   bool
	paused     bool
	over       bool
	startup    bool

	lastMove     time.Time
	lastPower    time.Time
	powerCycle   int
	moveInterval time.Duration

	currentCombo  int
	lastEatTime   time.Time
	comboFlashEnd time.Time
}

func scoreFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".snake_score")
}

func loadHighScore() int {
	data, err := os.ReadFile(scoreFile())
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return n
}

func saveHighScore(s int) {
	os.WriteFile(scoreFile(), []byte(strconv.Itoa(s)), 0644)
}

func NewGame(highScore int) *Game {
	g := &Game{
		highScore:  highScore,
		baseSpeed:  200 * time.Millisecond,
		startup:    true,
		powerCycle: 0,
	}
	g.init()
	return g
}

func (g *Game) init() {
	g.snake = nil
	midX, midY := MapW/2, MapH/2
	for i := 2; i >= 0; i-- {
		g.snake = append(g.snake, Point{midX - i, midY})
	}
	g.dir = Point{1, 0}
	g.nextDir = Point{1, 0}
	g.obstacles = nil
	g.foods = nil
	g.score = 0
	g.paused = false
	g.over = false
	g.speedBoost = false
	g.wallPass = false
	g.powerUp = nil
	g.powerCycle = 0
	g.moveInterval = g.baseSpeed
	g.currentCombo = 0
	g.lastEatTime = time.Time{}
	g.comboFlashEnd = time.Time{}

	occupied := map[Point]bool{}
	for _, s := range g.snake {
		occupied[s] = true
	}

	for len(g.obstacles) < NumObstacle {
		p := randPoint(1, MapW-2, 1, MapH-2)
		if !occupied[p] {
			g.obstacles = append(g.obstacles, p)
			occupied[p] = true
		}
	}

	for len(g.foods) < NumRedFood {
		p := g.randEmpty(occupied)
		g.foods = append(g.foods, p)
		occupied[p] = true
	}

	{
		p := g.randEmpty(occupied)
		g.goldFood = p
		occupied[p] = true
	}

	now := time.Now()
	g.lastMove = now
	g.lastPower = now
}

func randPoint(xmin, xmax, ymin, ymax int) Point {
	return Point{xmin + rand.Intn(xmax-xmin+1), ymin + rand.Intn(ymax-ymin+1)}
}

func (g *Game) randEmpty(occupied map[Point]bool) Point {
	for {
		p := randPoint(1, MapW-2, 1, MapH-2)
		if !occupied[p] {
			return p
		}
	}
}

func (g *Game) allOccupied() map[Point]bool {
	m := map[Point]bool{}
	for _, s := range g.snake {
		m[s] = true
	}
	for _, o := range g.obstacles {
		m[o] = true
	}
	for _, f := range g.foods {
		m[f] = true
	}
	m[g.goldFood] = true
	if g.powerUp != nil {
		m[g.powerUp.Pos] = true
	}
	for x := 0; x < MapW; x++ {
		m[Point{x, 0}] = true
		m[Point{x, MapH - 1}] = true
	}
	for y := 0; y < MapH; y++ {
		m[Point{0, y}] = true
		m[Point{MapW - 1, y}] = true
	}
	return m
}

func (g *Game) spawnFood(isGold bool) {
	occ := g.allOccupied()
	p := g.randEmpty(occ)
	if isGold {
		g.goldFood = p
	} else {
		for i, f := range g.foods {
			if occ[f] == false {
				g.foods[i] = p
				return
			}
		}
		g.foods = append(g.foods, p)
	}
}

func (g *Game) replaceRedFood(idx int) {
	occ := g.allOccupied()
	p := g.randEmpty(occ)
	g.foods[idx] = p
}

func (g *Game) replaceGoldFood() {
	occ := g.allOccupied()
	p := g.randEmpty(occ)
	g.goldFood = p
}

func (g *Game) spawnPowerUp() {
	if g.powerUp != nil {
		return
	}
	occ := g.allOccupied()
	p := g.randEmpty(occ)
	kind := PowerType(g.powerCycle % 3)
	g.powerCycle++
	g.powerUp = &PowerUp{
		Pos:       p,
		Kind:      kind,
		SpawnTime: time.Now(),
		Duration:  4 * time.Second,
	}
}

func (g *Game) updateCombo() int {
	now := time.Now()
	if !g.lastEatTime.IsZero() && now.Sub(g.lastEatTime) < 2*time.Second {
		g.currentCombo++
		if g.currentCombo > 5 {
			g.currentCombo = 5
		}
	} else {
		g.currentCombo = 1
	}
	g.lastEatTime = now
	g.comboFlashEnd = now.Add(300 * time.Millisecond)
	return g.currentCombo
}

func (g *Game) currentInterval() time.Duration {
	interval := g.baseSpeed
	now := time.Now()
	if g.speedBoost && now.Before(g.boostEnd) {
		interval /= 2
	} else {
		g.speedBoost = false
	}
	if !g.slowEnd.IsZero() && now.Before(g.slowEnd) {
		interval = interval * 3 / 2
	} else {
		g.slowEnd = time.Time{}
	}
	return interval
}

func (g *Game) move() bool {
	g.dir = g.nextDir
	head := g.snake[len(g.snake)-1]
	newHead := Point{head.X + g.dir.X, head.Y + g.dir.Y}

	if newHead.X <= 0 || newHead.X >= MapW-1 || newHead.Y <= 0 || newHead.Y >= MapH-1 {
		if g.wallPass {
			g.wallPass = false
			if newHead.X <= 0 {
				newHead.X = MapW - 2
			} else if newHead.X >= MapW-1 {
				newHead.X = 1
			}
			if newHead.Y <= 0 {
				newHead.Y = MapH - 2
			} else if newHead.Y >= MapH-1 {
				newHead.Y = 1
			}
		} else {
			return false
		}
	}

	for _, s := range g.snake {
		if s == newHead {
			return false
		}
	}
	for _, o := range g.obstacles {
		if o == newHead {
			return false
		}
	}

	g.snake = append(g.snake, newHead)

	ate := false
	for i, f := range g.foods {
		if f == newHead {
			mult := g.updateCombo()
			g.score += 10 * mult
			g.replaceRedFood(i)
			ate = true
			break
		}
	}
	if !ate && newHead == g.goldFood {
		mult := g.updateCombo()
		g.score += 30 * mult
		g.speedBoost = true
		g.boostEnd = time.Now().Add(5 * time.Second)
		g.replaceGoldFood()
		ate = true
	}

	if !ate {
		g.snake = g.snake[1:]
	}

	if g.powerUp != nil && newHead == g.powerUp.Pos {
		switch g.powerUp.Kind {
		case PowerSlow:
			g.slowEnd = time.Now().Add(8 * time.Second)
		case PowerWallPass:
			g.wallPass = true
		case PowerShrink:
			if len(g.snake) > 4 {
				g.snake = g.snake[3:]
			} else if len(g.snake) > 1 {
				g.snake = g.snake[len(g.snake)-1:]
			}
		}
		g.powerUp = nil
	}

	return true
}

func (g *Game) update() {
	now := time.Now()

	if g.powerUp != nil && now.Sub(g.powerUp.SpawnTime) >= g.powerUp.Duration {
		g.powerUp = nil
	}

	if now.Sub(g.lastPower) >= 12*time.Second {
		g.spawnPowerUp()
		g.lastPower = now
	}
}

func main() {
	highScore := loadHighScore()
	rand.New(rand.NewSource(time.Now().UnixNano()))

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating screen: %v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	screen.EnableMouse()
	screen.Clear()

	g := NewGame(highScore)
	r := NewRenderer()

	for {
		screen.Show()
		ev := screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyCtrlQ || ev.Rune() == 'q' {
				if g.score > g.highScore {
					g.highScore = g.score
					saveHighScore(g.highScore)
				}
				return
			}

			if g.startup {
				g.startup = false
				g.lastMove = time.Now()
				g.lastPower = time.Now()
				continue
			}

			if g.over {
				if ev.Rune() == 'r' || ev.Rune() == 'R' {
					g = NewGame(g.highScore)
					g.startup = false
					g.lastMove = time.Now()
					g.lastPower = time.Now()
				}
				continue
			}

			if ev.Rune() == 'p' || ev.Rune() == 'P' {
				g.paused = !g.paused
				if !g.paused {
					g.lastMove = time.Now()
				}
				continue
			}

			if g.paused {
				continue
			}

			switch ev.Key() {
			case tcell.KeyUp:
				if g.dir.Y != 1 {
					g.nextDir = Point{0, -1}
				}
			case tcell.KeyDown:
				if g.dir.Y != -1 {
					g.nextDir = Point{0, 1}
				}
			case tcell.KeyLeft:
				if g.dir.X != 1 {
					g.nextDir = Point{-1, 0}
				}
			case tcell.KeyRight:
				if g.dir.X != -1 {
					g.nextDir = Point{1, 0}
				}
			}

		case *tcell.EventResize:
			screen.Sync()
		}

		if !g.startup && !g.paused && !g.over {
			g.update()
			interval := g.currentInterval()
			if time.Since(g.lastMove) >= interval {
				if !g.move() {
					g.over = true
					if g.score > g.highScore {
						g.highScore = g.score
						saveHighScore(g.highScore)
					}
				}
				g.lastMove = time.Now()
			}
		}

		r.Draw(screen, g)
	}
}
