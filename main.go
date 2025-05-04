// go run example.go

package main

/*
#cgo LDFLAGS: -Wl,-rpath,. -ltermbox2

#include <stdint.h>
typedef struct tb_event_s {
    uint8_t _type;
    uint8_t mod;
    uint16_t key;
    uint32_t ch;
    int32_t w;
    int32_t h;
    int32_t x;
    int32_t y;
} tb_event;

int tb_init();
int tb_shutdown();

int tb_width();
int tb_height();

int tb_clear();
int tb_present();

int tb_set_cursor(int cx, int cy);
int tb_hide_cursor();

int tb_set_cell(int x, int y, uint32_t ch, uint32_t fg, uint32_t bg);

int tb_peek_event(tb_event *event, int timeout_ms);
int tb_poll_event(tb_event *event);

int tb_print(int x, int y, uint32_t fg, uint32_t bg, const char *str);

int tb_set_input_mode(int mode);
int tb_set_output_mode(int mode);
*/
import "C"
import (
    "time"
    "flag"
    "math/rand/v2"
)


type grid struct {
    grid [][]bool
    buffer [][]bool
}

func gridCreate(cfg *config) *grid {
    g := grid{}
    g.grid = make([][]bool, cfg.w)
    g.buffer = make([][]bool, cfg.w)
    for x := range cfg.w {
        g.grid[x] = make([]bool, cfg.h)
        g.buffer[x] = make([]bool, cfg.h)
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

func (g *grid) setRandomPattern(rand *rand.Rand, w, h, x, y int) {
    for px := x - w / 2; px < x + w / 2; px++ {
        for py := y - h / 2; py < y + h / 2; py++ {
            if g.oob(px, py) {
                continue
            }
            g.grid[px][py] = rand.IntN(2) == 1
        }
    }
}

func (g *grid) show(cfg *config) bool {
    cellOnScreen := false
    for x := range g.grid {
        for y := range g.grid[x] {
            if !g.grid[x][y] {
                continue
            }
            w := int(C.tb_width()) / 2
            h := int(C.tb_height())
            if x < cfg.x - w / 2 || x > cfg.x + w / 2 || y < cfg.y - h / 2 || y > cfg.y + h / 2 {
                continue
            }
            ns := g.neighbours(x, y)
            setCell(x, y, ns, cfg)
            cellOnScreen = true
        }
    }
    return cellOnScreen
}


func setCell(x, y, ns int, cfg *config) {
    var ch rune = rune(cfg.ch[0])
    if cfg.showNeighbours {
        ch = '0' + rune(ns)
    }

    sx := (x - cfg.x) * 2 + int(C.tb_width()) / 2
    sy := y - cfg.y + int(C.tb_height()) / 2
    C.tb_set_cell(C.int(sx),
                  C.int(sy),
                  C.uint32_t(ch),
                  C.uint32_t(cfg.fg),
                  C.uint32_t(cfg.bg))
    C.tb_set_cell(C.int(sx + 1),
                  C.int(sy),
                  C.uint32_t(ch),
                  C.uint32_t(cfg.fg), C.uint32_t(cfg.bg))
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
    w, h int
    x, y int
    paused bool
    speed int
    showNeighbours bool
    fg uint
    bg uint
    ch string
}

func configCreate() *config {
    w, h := 300, 300
    cfg := config{
        w: w,
        h: h,
        x: w / 2,
        y: h / 2,
    }

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

    // I := true
    // O := false
    // patterns := [][][]bool{
    //     {
    //         {O, I, O, O, O, O, O},
    //         {O, O, O, I, O, O, O},
    //         {I, I, O, O, I, I, I},
    //     },
    //     {
    //         {O, I, O},
    //         {I, I, I},
    //         {O, I, O},
    //     },
    //     {
    //         {O, O, I},
    //         {I, O, I},
    //         {O, I, I},
    //     },
    // }

    rand := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))

    C.tb_init()
    C.tb_set_output_mode(C.int(5)) // TB_OUTPUT_TRUECOLOR

    cfg := configCreate()
    g := gridCreate(cfg)
    stepTime := 0

    ev := C.tb_event{}

    for ev.key != 27 && ev.ch != 'q' {

        C.tb_peek_event(&ev, 0)

        // C.tb_print(C.int(0), C.int(1), C.uint(0xffffff), C.uint(0x0000ff), C.CString(*s))
        switch ev.ch {
        case ' ':
            cfg.paused = !cfg.paused
        case '-':
            cfg.speed = max(1, cfg.speed - 1)
        case '+':
            cfg.speed = min(10, cfg.speed + 1)
        case 'r':
            cfg.x = cfg.w / 2
            cfg.y = cfg.h / 2
            g.clear()
        case 'w':
            cfg.y -= 1
        case 's':
            cfg.y += 1
        case 'a':
            cfg.x -= 1
        case 'd':
            cfg.x += 1
        default:
            break
        }

        if !g.show(cfg) {
            g.clear()
            // g.setPattern(patterns[rand.IntN(len(patterns))], cfg.x, cfg.y)
            g.setRandomPattern(rand, 10, 10, cfg.x, cfg.y)
        }
        C.tb_present()
        C.tb_clear()

        if !cfg.paused && stepTime > 10 - cfg.speed {
            g.step()
            stepTime = 0
        }
        stepTime += 1

        time.Sleep(40 * time.Millisecond)
    }

    C.tb_shutdown()
}
