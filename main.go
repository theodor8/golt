package main

/*
#cgo LDFLAGS: -L${SRCDIR}/termbox2 -Wl,-rpath,${SRCDIR}/termbox2 -l:libtermbox2.a


#define TB_IMPL
#define TB_OPT_ATTR_W 32

#include <stdlib.h>
#include <stdint.h>

typedef uint32_t uintattr_t;

struct tb_event {
    uint8_t type; // one of `TB_EVENT_*` constants
    uint8_t mod;  // bitwise `TB_MOD_*` constants
    uint16_t key; // one of `TB_KEY_*` constants
    uint32_t ch;  // a Unicode codepoint
    int32_t w;    // resize width
    int32_t h;    // resize height
    int32_t x;    // mouse x
    int32_t y;    // mouse y
};

int tb_init();
int tb_shutdown();
int tb_width();
int tb_height();
int tb_clear();
int tb_present();
int tb_set_cursor(int cx, int cy);
int tb_hide_cursor();
int tb_set_cell(int x, int y, uint32_t ch, uintattr_t fg, uintattr_t bg);
int tb_peek_event(struct tb_event *event, int timeout_ms);
int tb_poll_event(struct tb_event *event);
int tb_print(int x, int y, uintattr_t fg, uintattr_t bg, const char *str);
int tb_printf(int x, int y, uintattr_t fg, uintattr_t bg, const char *fmt, ...);
int tb_set_output_mode(int mode);

*/
import "C"
import (
	"flag"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
	"unsafe"
)



type cell struct {
    val bool
    ns, age uint8
}
type grid [][]cell

func gridCreate(w, h int) grid {
    var g grid = make([][]cell, w)
    for x := range g {
        g[x] = make([]cell, h)
    }
    return g
}

func (g grid) oob(x, y int) bool {
    return x < 0 || x >= len(g) || y < 0 || y >= len(g[0])
}

func (g grid) clear() {
    for x := range g {
        for y := range g[x] {
            g[x][y] = cell{}
        }
    }
}

func (g grid) resize(w, h int) grid {
    if w <= 10 || h <= 10 {
        return g
    }
    if w != len(g) {
        if w > cap(g) {
            c := cap(g)
            g = slices.Grow(g, w - cap(g))
            g = g[:w]
            for x := c; x < w; x++ {
                g[x] = make([]cell, len(g[0]), cap(g[0]))
            }
        }
        g = g[:w]
    }
    if h != len(g[0]) {
        if h > cap(g[0]) {
            c := cap(g[0])
            for x := range g {
                g[x] = slices.Grow(g[x], h - c)
            }
        }
        for x := range g {
            g[x] = g[x][:h]
        }
    }
    return g
}




func (g grid) setPattern(pattern [][]bool, x, y int) {
    for py := range pattern {
        for px := range pattern[py] {
            if !pattern[py][px] || g.oob(px + x, py + y) {
                continue
            }
            g[px + x][py + y].val = true
        }
    }
}

func (g grid) setRandomPattern(rand *rand.Rand, x, y, w, h, n int) {
    for px := x - w / 2; px < x + w / 2; px++ {
        for py := y - h / 2; py < y + h / 2; py++ {
            if g.oob(px, py) {
                continue
            }
            g[px][py].val = rand.IntN(n) == 1
        }
    }
}

func (g grid) show(cfg *config) {
    for x := range g {
        for y := range g[x] {
            if !g[x][y].val {
                continue
            }

            var lch rune = rune(cfg.cellText[0])
            var rch rune = rune(cfg.cellText[1])

            if cfg.showNeighbours {
                ns := fmt.Sprintf("%02d", g[x][y].ns % 100)
                lch = rune(ns[0])
                rch = rune(ns[1])
            }

            if cfg.showAge {
                age := fmt.Sprintf("%02d", g[x][y].age % 100)
                lch = rune(age[0])
                rch = rune(age[1])
            }

            sx := x * 2
            sy := y
            C.tb_set_cell(C.int(sx), C.int(sy), C.uint32_t(lch), C.uintattr_t(cfg.fgColor), C.uintattr_t(cfg.bgColor))
            C.tb_set_cell(C.int(sx + 1), C.int(sy), C.uint32_t(rch), C.uintattr_t(cfg.fgColor), C.uintattr_t(cfg.bgColor))
        }
    }
}


func (g grid) computeNeighbours(wrap bool) {
    for x := range g {
        for y := range g[x] {
            var ns uint8 = 0
            for dx := -1; dx < 2; dx++ {
                nx := x + dx
                if wrap {
                    nx = (nx + len(g)) % len(g)
                }
                for dy := -1; dy < 2; dy++ {
                    ny := y + dy
                    if wrap {
                        ny = (ny + len(g[0])) % len(g[0])
                    } else if g.oob(nx, ny) {
                        continue
                    }
                    if (dx == 0 && dy == 0) {
                        continue
                    }
                    if g[nx][ny].val {
                        ns += 1
                    }
                }
            }
            g[x][y].ns = ns

        }
    }
}

func (g grid) step(wrap bool) {
    for x := range g {
        for y := range g[x] {
            if g[x][y].val {
                if g[x][y].ns < 2 || g[x][y].ns > 3 {
                    g[x][y].val = false
                    g[x][y].age = 0
                } else {
                    g[x][y].age += 1
                }
            } else if g[x][y].ns == 3 {
                g[x][y].val = true
                g[x][y].age = 1
            }
        }
    }
    g.computeNeighbours(wrap)
}


type config struct {
    paused bool
    speed int
    debug bool
    showNeighbours bool
    fgColor uint
    bgColor uint
    cellText string
    wrap bool
    showAge bool
}

func configCreate() *config {
    cfg := config{}

    flag.BoolVar(&cfg.paused, "p", false, "Paused")
    flag.IntVar(&cfg.speed, "s", 8, "Speed (steps per second)")
    flag.BoolVar(&cfg.debug, "d", false, "Debug mode")
    flag.BoolVar(&cfg.showNeighbours, "n", false, "Show number of cell neighbours")
    flag.UintVar(&cfg.fgColor, "f", 0x0000ff, "Foreground (text) color")
    flag.UintVar(&cfg.bgColor, "b", 0xffffff, "Background color")
    flag.StringVar(&cfg.cellText, "t", "  ", "Text to use for cells, length 2")
    flag.BoolVar(&cfg.wrap, "w", false, "Wrap around")
    flag.BoolVar(&cfg.showAge, "a", false, "Show age of cells")

    flag.Parse()

    if len(cfg.cellText) != 2 {
        cfg.cellText = "  "
    }

    cfg.speed = max(1, cfg.speed)

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

    rand := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))

    C.tb_init()
    defer C.tb_shutdown()
    C.tb_set_output_mode(C.int(5)) // TB_OUTPUT_TRUECOLOR

    cfg := configCreate()
    g := gridCreate(int(C.tb_width()) / 2, int(C.tb_height()))

    // g.setPattern(patterns[rand.IntN(len(patterns))], len(g) / 2, len(g[0]) / 2)
    g.setPattern(patterns[2], len(g) / 2, len(g[0]) / 2)

    ev := C.struct_tb_event{}
    var stepDuration time.Duration
    frameTime := time.Now()

    g.computeNeighbours(cfg.wrap) // for first step and render
    for ev.key != 27 && ev.ch != 'q' {

        g = g.resize(int(C.tb_width()) / 2, int(C.tb_height()))
        C.tb_peek_event(&ev, 0)

        if cfg.debug {
            s := C.CString(fmt.Sprintf("tbw=%v tbh=%v gw=%v gh=%v s=%v p=%v", C.tb_width(), C.tb_height(), len(g), len(g[0]), cfg.speed, cfg.paused))
            C.tb_print(0, 0, 0x01ffffff, 0, s)
            C.free(unsafe.Pointer(s))
        }

        switch ev.ch {
        case ' ':
            cfg.paused = !cfg.paused
        case '-':
            cfg.speed = max(1, cfg.speed - 1)
        case '+':
            cfg.speed += 1 
        case 'r':
            g.clear()
            g.setRandomPattern(rand, len(g) / 2, len(g[0]) / 2, 30, 30, 2)
            g.computeNeighbours(cfg.wrap)
        default:
            break
        }


        g.show(cfg)

        C.tb_present()
        C.tb_clear()

        if !cfg.paused {
            stepDuration += time.Since(frameTime)
            for stepDuration >= time.Second / time.Duration(cfg.speed) {
                g.step(cfg.wrap)
                stepDuration -= time.Second / time.Duration(cfg.speed)
            }
        }
        frameTime = time.Now()

        time.Sleep(time.Millisecond * 40)
    }

}
