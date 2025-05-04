package main

/*
#define TB_IMPL
#define TB_OPT_ATTR_W 32
#include "termbox2.h"
*/
import "C"
import (
	"flag"
	"math/rand/v2"
	"slices"
	"time"
)


type grid struct {
    grid [][]bool
    buffer [][]bool
}

func gridCreate(w, h int) *grid {
    g := grid{}
    g.grid = make([][]bool, w)
    g.buffer = make([][]bool, w)
    for x := range w {
        g.grid[x] = make([]bool, h)
        g.buffer[x] = make([]bool, h)
    }
    return &g
}

func (g *grid) oob(x, y int) bool {
    return x < 0 || x >= len(g.grid) || y < 0 || y >= len(g.grid[0])
}

func (g *grid) clear() {
    for x := range g.grid {
        for y := range g.grid[x] {
            g.grid[x][y] = false
        }
    }
}

func (g *grid) resize(w, h int) {
    if w != len(g.grid) {
        if w > cap(g.grid) {
            g.grid = slices.Grow(g.grid, (w - cap(g.grid)) * 2)
            g.buffer = slices.Grow(g.buffer, (w - cap(g.grid)) * 2)
        }
        g.grid = g.grid[:w]
        g.buffer = g.buffer[:w]
    }
    if h != len(g.grid[0]) {
        if h > cap(g.grid[0]) {
            for x := range g.grid {
                g.grid[x] = slices.Grow(g.grid[x], (h - cap(g.grid[0])) * 2)
                g.buffer[x] = slices.Grow(g.buffer[x], (h - cap(g.grid[0])) * 2)
            }
        }
        for x := range g.grid {
            g.grid[x] = g.grid[x][:h]
            g.buffer[x] = g.buffer[x][:h]
        }
    }
}




func (g *grid) setPattern(pattern [][]bool, x, y int) {
    for py := range pattern {
        for px := range pattern[py] {
            if !pattern[py][px] || g.oob(px + x, py + y) {
                continue
            }
            g.grid[px + x][py + y] = true
        }
    }
}

func (g *grid) setRandomPattern(rand *rand.Rand, x, y, w, h int) {
    for px := x - w / 2; px < x + w / 2; px++ {
        for py := y - h / 2; py < y + h / 2; py++ {
            if g.oob(px, py) {
                continue
            }
            g.grid[px][py] = rand.IntN(2) == 1
        }
    }
}

func (g *grid) show(cfg *config) {
    for x := range g.grid {
        for y := range g.grid[x] {
            if !g.grid[x][y] {
                continue
            }

            var ch rune = rune(cfg.ch[0])
            if cfg.showNeighbours {
                ch = '0' + rune(g.neighbours(x, y))
            }

            sx := x * 2
            sy := y
            C.tb_set_cell(C.int(sx), C.int(sy), C.uint32_t(ch), C.uintattr_t(cfg.fg), C.uintattr_t(cfg.bg))
            C.tb_set_cell(C.int(sx + 1), C.int(sy), C.uint32_t(ch), C.uintattr_t(cfg.fg), C.uintattr_t(cfg.bg))
        }
    }
}



func (g *grid) neighbours(x, y int) int {
    ns := 0
    for dx := -1; dx < 2; dx++ {
        nx := x + dx
        if nx < 0 || nx >= len(g.grid) {
            continue
        }
        for dy := -1; dy < 2; dy++ {
            ny := y + dy
            if (dx == 0 && dy == 0) || ny < 0 || ny >= len(g.grid[0]) {
                continue
            }
            if g.grid[nx][ny] {
                ns += 1
            }
        }
    }
    return ns
}

func (g *grid) step() {
    for x := range g.grid {
        for y := range g.grid[x] {
            g.buffer[x][y] = g.grid[x][y]
            ns := g.neighbours(x, y)
            if g.grid[x][y] {
                if ns < 2 || ns > 3 {
                    g.buffer[x][y] = false
                }
            } else if ns == 3 {
                g.buffer[x][y] = true
            }
        }
    }
    g.grid, g.buffer = g.buffer, g.grid
}



type config struct {
    paused bool
    speed int
    showNeighbours bool
    fg uint
    bg uint
    ch string
}

func configCreate() *config {
    cfg := config{}

    flag.BoolVar(&cfg.showNeighbours, "sn", false, "Show neighbours")
    flag.IntVar(&cfg.speed, "s", 5, "Speed (1-10)")
    flag.BoolVar(&cfg.paused, "p", false, "Paused")
    flag.UintVar(&cfg.fg, "fg", 0x0000ff, "Foreground (character) color")
    flag.UintVar(&cfg.bg, "bg", 0xffffff, "Background color")
    flag.StringVar(&cfg.ch, "ch", " ", "Character to use")

    flag.Parse()

    return &cfg
}


func main() {

    I := true
    O := false
    patterns := [][][]bool{
        {
            {O, I, O, O, O, O, O},
            {O, O, O, I, O, O, O},
            {I, I, O, O, I, I, I},
        },
        {
            {O, I, O},
            {I, I, I},
            {O, I, O},
        },
        {
            {O, O, I},
            {I, O, I},
            {O, I, I},
        },
    }

    // rand := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))

    C.tb_init()
    C.tb_set_output_mode(C.int(5)) // TB_OUTPUT_TRUECOLOR

    cfg := configCreate()
    g := gridCreate(int(C.tb_width()) / 2, int(C.tb_height()))
    stepTime := 0

    // g.setPattern(patterns[rand.IntN(len(patterns))], cfg.x, cfg.y)
    // g.setPattern(patterns[2], len(g.grid) / 2, len(g.grid[0]) / 2)
    g.setPattern(patterns[2], 0, 0)
    // g.setRandomPattern(rand, len(g.grid) / 2, len(g.grid[0]) / 2, 5, 5)

    ev := C.struct_tb_event{}

    for ev.key != 27 && ev.ch != 'q' {

        C.tb_peek_event(&ev, 0)

        // C.tb_print(C.int(0), C.int(0), C.uint(0xffffff), C.uint(0x0000ff), C.CString(fmt.Sprintf("tbw=%d tbh=%d gw=%d gh=%d", C.tb_width(), C.tb_height(), len(g.grid), len(g.grid[0]))))

        switch ev.ch {
        case ' ':
            cfg.paused = !cfg.paused
        case '-':
            cfg.speed = max(1, cfg.speed - 1)
        case '+':
            cfg.speed = min(10, cfg.speed + 1)
        case 'r':
            g.clear()
        default:
            break
        }

        g.show(cfg)


        C.tb_present()
        C.tb_clear()

        if !cfg.paused && stepTime > 10 - cfg.speed {
            g.step()
            stepTime = 0
        }
        stepTime += 1

        g.resize(int(C.tb_width()) / 2, int(C.tb_height()))

        time.Sleep(40 * time.Millisecond)
    }

    C.tb_shutdown()
}
