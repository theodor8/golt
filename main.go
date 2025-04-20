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
	"fmt"
	"time"
)


type grid struct {
    grid [][]bool
    buffer [][]bool
}


func grid_create(w, h int) grid {
    g := grid{}
    g.grid = make([][]bool, w)
    g.buffer = make([][]bool, w)
    for x := range w {
        g.grid[x] = make([]bool, h)
        g.buffer[x] = make([]bool, h)
    }
    return g
}

func (g grid) show(dx, dy int) {
    for x := range g.grid {
        for y := range g.grid[x] {
            if !g.grid[x][y] {
                continue
            }
            ns := g.neighrbours(x, y)
            C.tb_set_cell(C.int((x - dx) * 2), C.int(y - dy), C.uint32_t('0' + ns), C.uint32_t(0x0000ff), C.uint32_t(0xffffff))
            C.tb_set_cell(C.int((x - dx) * 2 + 1), C.int(y - dy), C.uint32_t(' '), C.uint32_t(0x000000), C.uint32_t(0xffffff))
        }
    }
}

func (g grid) neighrbours(x, y int) int {
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
            ns := g.neighrbours(x, y)
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



func main() {

    C.tb_init()
    C.tb_set_input_mode(C.int(0b101)) // TB_INPUT_MOUSE
    C.tb_set_output_mode(C.int(5)) // TB_OUTPUT_TRUECOLOR

    const width, height = 500, 500
    g := grid_create(width, height)
    dx, dy := width / 2, height / 2
    stepDelay := 20
    paused := false
    leftMouseDown := false
    rightMouseDown := false
    stepTime := 0

    lastEv := C.tb_event{}
    ev := C.tb_event{}

    for ev.key != 27 {
        C.tb_peek_event(&ev, 0)
        C.tb_print(C.int(0), C.int(0), C.uint(0xffffff), C.uint(0x0000ff), C.CString(fmt.Sprintf(
            "event: type=%d mod=%d key=%d ch=%d w=%d h=%d x=%d y=%d",
            ev._type,
            ev.mod,
            ev.key,
            ev.ch,
            ev.w,
            ev.h,
            ev.x,
            ev.y)))
        C.tb_print(C.int(0), C.int(1), C.uint(0xffffff), C.uint(0x0000ff), C.CString(fmt.Sprintf(
            "dx=%v dy=%v speed=%v paused=%v", dx, dy, stepDelay, paused)))

        switch ev.key {
        case 65512: // left mouse down
            if !leftMouseDown {
                leftMouseDown = true
                g.grid[ev.x / 2 + C.int(dx)][ev.y + C.int(dy)] = true
            }
        case 65511: // right mouse down
            if !rightMouseDown {
                rightMouseDown= true
            } else {
                dx -= int(ev.x - lastEv.x) / 2
                dy -= int(ev.y - lastEv.y)
            }
        // case 65509: // mouse release
        case 65514: // right
            dx += 1
        case 65515: // left
            dx -= 1
        case 65516: // down
            dy += 1
        case 65517: // up
            dy -= 1
        default:
            leftMouseDown = false
            rightMouseDown = false
        }
        switch ev.ch {
        case 32:
            paused = !paused
        case 45: // minus
            stepDelay += 1
        case 43: // plus
            stepDelay -= 1
        default:
            break
        }

        if dx < 0 {
            dx = 0
        } else if dx > (width - 1 - int(C.tb_width()) / 2) {
            dx = (width - 1 - int(C.tb_width()) / 2)
        }
        if dy < 0 {
            dy = 0
        } else if dy > height - 1 - int(C.tb_height()) {
            dy = height - 1 - int(C.tb_height())
        }
        if stepDelay < 0 {
            stepDelay = 0
        } else if stepDelay > 1000 {
            stepDelay = 1000
        }

        g.show(dx, dy)
        C.tb_present()
        C.tb_clear()

        if !paused && stepTime > stepDelay {
            g.step()
            stepTime = 0
        }
        stepTime += 1
        lastEv = ev

        time.Sleep(10 * time.Millisecond)
    }

    C.tb_shutdown()
}
