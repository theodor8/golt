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
    "math/rand/v2"
    "slices"
    "time"
    "fmt"
    "unsafe"
)


type grid struct {
    grid [][]bool
    buffer [][]bool
    ns [][]int // neighbours
}

func gridCreate(w, h int) *grid {
    g := grid{}
    g.grid = make([][]bool, w)
    g.buffer = make([][]bool, w)
    g.ns = make([][]int, w)
    for x := range w {
        g.grid[x] = make([]bool, h)
        g.buffer[x] = make([]bool, h)
        g.ns[x] = make([]int, h)
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
    if w <= 10 || h <= 10 {
        return
    }
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

            var lch rune = rune(cfg.cellText[0])
            var rch rune = rune(cfg.cellText[1])
            if cfg.showNeighbours {
                lch = '0' + rune(g.ns[x][y])
                rch = '0' + rune(g.ns[x][y])

            }

            sx := x * 2
            sy := y
            C.tb_set_cell(C.int(sx), C.int(sy), C.uint32_t(lch), C.uintattr_t(cfg.fgColor), C.uintattr_t(cfg.bgColor))
            C.tb_set_cell(C.int(sx + 1), C.int(sy), C.uint32_t(rch), C.uintattr_t(cfg.fgColor), C.uintattr_t(cfg.bgColor))
        }
    }
}


func (g *grid) computeNeighbours(wrap bool) {
    for x := range g.grid {
        for y := range g.grid[x] {
            ns := 0
            for dx := -1; dx < 2; dx++ {
                nx := x + dx
                if wrap {
                    nx = (nx + len(g.grid)) % len(g.grid)
                }
                for dy := -1; dy < 2; dy++ {
                    ny := y + dy
                    if wrap {
                        ny = (ny + len(g.grid[0])) % len(g.grid[0])
                    } else if g.oob(nx, ny) {
                        continue
                    }
                    if (dx == 0 && dy == 0) {
                        continue
                    }
                    if g.grid[nx][ny] {
                        ns += 1
                    }
                }
            }
            g.ns[x][y] = ns
        }
    }
}

func (g *grid) step() {
    for x := range g.grid {
        for y := range g.grid[x] {
            g.buffer[x][y] = g.grid[x][y]
            if g.grid[x][y] {
                if g.ns[x][y] < 2 || g.ns[x][y] > 3 {
                    g.buffer[x][y] = false
                }
            } else if g.ns[x][y] == 3 {
                g.buffer[x][y] = true
            }
        }
    }
    g.grid, g.buffer = g.buffer, g.grid
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
}

func configCreate() *config {
    cfg := config{}

    flag.BoolVar(&cfg.paused, "p", false, "Paused")
    flag.IntVar(&cfg.speed, "s", 5, "Speed (1-10)")
    flag.BoolVar(&cfg.debug, "d", false, "Debug mode")
    flag.BoolVar(&cfg.showNeighbours, "n", false, "Show neighbours")
    flag.UintVar(&cfg.fgColor, "f", 0x0000ff, "Foreground (text) color")
    flag.UintVar(&cfg.bgColor, "b", 0xffffff, "Background color")
    flag.StringVar(&cfg.cellText, "t", "  ", "Text to use for cells, length 2")
    flag.BoolVar(&cfg.wrap, "w", false, "Wrap around")

    flag.Parse()

    if len(cfg.cellText) != 2 {
        cfg.cellText = "  "
    }

    cfg.speed = min(max(1, cfg.speed), 10)

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
    stepTime := 0

    // g.setPattern(patterns[rand.IntN(len(patterns))], len(g.grid) / 2, len(g.grid[0]) / 2)
    g.setPattern(patterns[2], len(g.grid) / 2, len(g.grid[0]) / 2)

    ev := C.struct_tb_event{}

    for ev.key != 27 && ev.ch != 'q' {

        C.tb_peek_event(&ev, 0)

        if cfg.debug {
            s := C.CString(fmt.Sprintf("tbw=%v tbh=%v gw=%v gh=%v s=%v p=%v", C.tb_width(), C.tb_height(), len(g.grid), len(g.grid[0]), cfg.speed, cfg.paused))
            C.tb_print(0, 0, 0x01ffffff, 0, s)
            C.free(unsafe.Pointer(s))
        }

        switch ev.ch {
        case ' ':
            cfg.paused = !cfg.paused
        case '-':
            cfg.speed = max(1, cfg.speed - 1)
        case '+':
            cfg.speed = min(10, cfg.speed + 1)
        case 'r':
            g.clear()
            g.setRandomPattern(rand, len(g.grid) / 2, len(g.grid[0]) / 2, 10, 10)
        default:
            break
        }

        g.computeNeighbours(cfg.wrap)

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

}
