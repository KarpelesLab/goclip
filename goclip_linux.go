package goclip

/*
#cgo pkg-config: xcb xcb-atom xcb-event xcb-icccm xcb-xfixes

#include <stdlib.h>
#include <xcb/xcb.h>
#include <xcb/xcb_atom.h>
#include <xcb/xcb_event.h>
#include <xcb/xcb_icccm.h>
#include <xcb/xcb_aux.h>
#include <xcb/xfixes.h>

xcb_screen_t *screen_of_display (xcb_connection_t *c, int screen) {
	xcb_screen_iterator_t iter;

	iter = xcb_setup_roots_iterator (xcb_get_setup (c));
	for (; iter.rem; --screen, xcb_screen_next (&iter))
		if (screen == 0)
			return iter.data;

	return NULL;
}
*/
import "C"

import (
	"context"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

// internally used to forward events through channels
type evData struct {
	selection C.xcb_atom_t
	property  C.xcb_atom_t
	target    C.xcb_atom_t
}

// represents one value in clipboard (can become invalid if not used quick enough)
type atom struct {
	name  string
	value C.xcb_atom_t // uint32
	board Board
}

type internal struct {
	dpy *C.xcb_connection_t
	win C.xcb_window_t
	op  sync.Once
	mon []*Monitor

	atoms   map[string]C.xcb_atom_t // C.xcb_atom_t is an alias of uint32
	atomsLk sync.RWMutex

	query_ext *C.xcb_query_extension_reply_t

	expectEv  map[Board]chan evData
	expectEvL sync.RWMutex

	copyVal  map[Board]Data
	copyValL sync.RWMutex
}

var fmtTypes = map[string]Type{
	"UTF8_STRING":                  Text,
	"text/plain;charset=utf-8":     Text,
	"STRING":                       Text,
	"TEXT":                         Text,
	"text/plain":                   Text,
	"image/png":                    Image,
	"image/bmp":                    Image,
	"image/x-bmp":                  Image,
	"image/x-MS-bmp":               Image,
	"image/x-win-bitmap":           Image,
	"image/tiff":                   Image,
	"image/jpeg":                   Image,
	"text/uri-list":                FileList,
	"x-special/gnome-copied-files": FileList,
}

func guessType(l []atom) Type {
	for _, a := range l {
		if t, ok := fmtTypes[a.name]; ok {
			return t
		}
	}
	return Invalid
}

func doInit() *internal {
	// do not do anything here, instead we connect at the first use of any method
	return &internal{
		atoms:    make(map[string]C.xcb_atom_t),
		expectEv: make(map[Board]chan evData),
		copyVal:  make(map[Board]Data),
	}
}

// boardEvChan returns a channel for a given board, creating it if needed
func (i *internal) boardEvChan(b Board) chan evData {
	i.expectEvL.RLock()
	ch, ok := i.expectEv[b]
	i.expectEvL.RUnlock()

	if ok {
		return ch
	}

	i.expectEvL.Lock()
	defer i.expectEvL.Unlock()

	// re-check, in case of race condition
	if ch, ok = i.expectEv[b]; ok {
		return ch
	}

	ch = make(chan evData, 4)
	i.expectEv[b] = ch
	return ch
}

func (i *internal) open() {
	log.Printf("goclip: creating new connection to X11 ...")

	var wg sync.WaitGroup
	wg.Add(1)
	go i.run(&wg)
	wg.Wait()
}

func (i *internal) paste(ctx context.Context, board Board) (Data, error) {
	i.op.Do(i.open)
	atom, found := i.atomCk(linuxBoardName(board))
	if !found {
		return nil, os.ErrNotExist
	}

	ch := i.boardEvChan(board)
	C.xcb_convert_selection(i.dpy, i.win, atom, i.atom("TARGETS"), i.atom("FOO"), C.XCB_CURRENT_TIME)
	C.xcb_flush(i.dpy)

	// TODO check property

	select {
	case sEv := <-ch:
		if sEv.property == 0 {
			return nil, os.ErrNotExist
		}
		data := i.spawnData(sEv.selection, sEv.property)
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (i *internal) copy(ctx context.Context, board Board, value Data) error {
	i.op.Do(i.open)
	if i.win == 0 {
		// not available
		return ErrNoSys
	}

	// ok let's do this
	atom, ok := i.atomCk(linuxBoardName(board))
	if !ok {
		return ErrNoBoard
	}

	if value.Type() == Invalid {
		// special case
		C.xcb_set_selection_owner(i.dpy, C.XCB_NONE, atom, C.XCB_CURRENT_TIME)
		i.copyValL.Lock()
		defer i.copyValL.Unlock()

		i.copyVal[board] = nil
		return nil
	}

	log.Printf("goclip: set self owner of selection %s for %s", value, board)
	i.copyValL.Lock()
	defer i.copyValL.Unlock()
	i.copyVal[board] = value
	C.xcb_set_selection_owner_checked(i.dpy, i.win, atom, C.XCB_CURRENT_TIME)
	C.xcb_flush(i.dpy)
	//res := C.xcb_get_selection_owner_reply(i.dpy, cookie, nil)
	//log.Printf("result %+v", res)
	return nil
}

func (i *internal) fetch(ctx context.Context, b Board, format C.xcb_atom_t) ([]byte, error) {
	selection, ok := i.atomCk(linuxBoardName(b))
	if !ok {
		return nil, os.ErrNotExist
	}

	ch := i.boardEvChan(b)
	C.xcb_convert_selection(i.dpy, i.win, selection, format, i.atom("FOO"), C.XCB_CURRENT_TIME)
	C.xcb_flush(i.dpy)

	select {
	case sEv := <-ch:
		//log.Printf("received fetch data %+v", sEv)
		// data is here
		var buf []byte
		offset := C.uint32_t(0)

		for {
			reply := C.xcb_get_property_reply(i.dpy, C.xcb_get_property(i.dpy, 1, i.win, sEv.property, format, offset, 16384), nil)
			tmp := C.GoBytes(C.xcb_get_property_value(reply), C.xcb_get_property_value_length(reply))
			//log.Printf("performed one read, len=%d bytes_after=%d all=%+v", C.xcb_get_property_value_length(reply), reply.bytes_after, reply)
			buf = append(buf, tmp...)
			if reply.bytes_after > 0 {
				offset += C.uint32_t(len(tmp)) / 4
				C.free(unsafe.Pointer(reply))
				continue
			}
			C.free(unsafe.Pointer(reply))
			return buf, nil
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (i *internal) monitor(mon *Monitor) error {
	i.op.Do(i.open)
	i.mon = append(i.mon, mon)
	return nil
}

func (i *internal) unmonitor(mon *Monitor) error {
	// locate & remove from i.mon
	for n, v := range i.mon {
		if v == mon {
			i.mon = append(i.mon[:n], i.mon[n+1:]...)
			return nil
		}
	}
	return os.ErrNotExist
}

func (i *internal) poll(mon *Monitor) error {
	return nil
}

func (i *internal) atom(s string) C.xcb_atom_t {
	v, _ := i.atomCk(s)
	return v
}

func (i *internal) atomCk(s string) (C.xcb_atom_t, bool) {
	i.atomsLk.RLock()
	a, ok := i.atoms[s]
	i.atomsLk.RUnlock()
	if ok {
		return a, true
	}

	if s == "" {
		return 0, false
	}

	// fetch atom
	cstr := C.CString(s)
	rep := C.xcb_intern_atom_reply(i.dpy, C.xcb_intern_atom(i.dpy, 0, C.ushort(len(s)), cstr), nil)
	a = rep.atom
	C.free(unsafe.Pointer(rep))
	C.free(unsafe.Pointer(cstr))

	// store in cache
	i.atomsLk.Lock()
	defer i.atomsLk.Unlock()
	i.atoms[s] = a
	return a, true
}

func (i *internal) resolveAtom(a C.xcb_atom_t) string {
	i.atomsLk.RLock()
	for s, sa := range i.atoms {
		if sa == a {
			i.atomsLk.RUnlock()
			return s
		}
	}
	i.atomsLk.RUnlock()

	reply2 := C.xcb_get_atom_name_reply(i.dpy, C.xcb_get_atom_name(i.dpy, a), nil)
	defer C.free(unsafe.Pointer(reply2))
	ln := C.xcb_get_atom_name_name_length(reply2)
	nm := C.xcb_get_atom_name_name(reply2)
	s := C.GoStringN(nm, ln)

	i.atomsLk.Lock()
	defer i.atomsLk.Unlock()
	i.atoms[s] = a
	return s
}

func (i *internal) run(wg *sync.WaitGroup) {
	runtime.LockOSThread()
	var defaultScreen C.int

	i.dpy = C.xcb_connect(nil, &defaultScreen)
	if i.dpy == nil {
		wg.Done()
		return
	}
	if errNo := C.xcb_connection_has_error(i.dpy); errNo != 0 {
		log.Printf("goclip: failed to connect to xcb: error #%d", errNo)
		return
	}
	i.query_ext = C.xcb_get_extension_data(i.dpy, &C.xcb_xfixes_id)
	if i.query_ext == nil {
		log.Printf("goclip: failed to find xfixes")
	}

	// let's cache our atoms
	for _, s := range []string{"UTF8_STRING", "CLIPBOARD", "PRIMARY", "SECONDARY", "TARGETS", "STRING", "TEXT", "FOO"} {
		i.atom(s)
	}

	screen := C.screen_of_display(i.dpy, defaultScreen)
	if screen == nil {
		// failed
		return
	}
	selection_window := C.xcb_generate_id(i.dpy)
	mask := C.uint(C.XCB_CW_BACK_PIXEL | C.XCB_CW_OVERRIDE_REDIRECT | C.XCB_CW_EVENT_MASK)
	values := []C.uint{screen.black_pixel, 1, C.XCB_EVENT_MASK_PROPERTY_CHANGE, 0}
	C.xcb_create_window(i.dpy, screen.root_depth, selection_window, screen.root, -10, -10, 1, 1, 0, C.XCB_COPY_FROM_PARENT, screen.root_visual, mask, unsafe.Pointer(&values[0]))
	C.xcb_discard_reply(i.dpy, C.xcb_xfixes_query_version(i.dpy, 1, 0).sequence)

	C.xcb_xfixes_select_selection_input(i.dpy, selection_window, i.atom("CLIPBOARD"), C.XCB_XFIXES_SELECTION_EVENT_MASK_SET_SELECTION_OWNER|C.XCB_XFIXES_SELECTION_EVENT_MASK_SELECTION_WINDOW_DESTROY|C.XCB_XFIXES_SELECTION_EVENT_MASK_SELECTION_CLIENT_CLOSE)
	C.xcb_xfixes_select_selection_input(i.dpy, selection_window, i.atom("PRIMARY"), C.XCB_XFIXES_SELECTION_EVENT_MASK_SET_SELECTION_OWNER|C.XCB_XFIXES_SELECTION_EVENT_MASK_SELECTION_WINDOW_DESTROY|C.XCB_XFIXES_SELECTION_EVENT_MASK_SELECTION_CLIENT_CLOSE)
	C.xcb_xfixes_select_selection_input(i.dpy, selection_window, i.atom("SECONDARY"), C.XCB_XFIXES_SELECTION_EVENT_MASK_SET_SELECTION_OWNER|C.XCB_XFIXES_SELECTION_EVENT_MASK_SELECTION_WINDOW_DESTROY|C.XCB_XFIXES_SELECTION_EVENT_MASK_SELECTION_CLIENT_CLOSE)

	class := "goclip\x00goclip"
	C.xcb_icccm_set_wm_class(i.dpy, selection_window, C.uint(len(class)), C.CString(class))

	C.xcb_icccm_set_wm_name(i.dpy, selection_window, C.XCB_ATOM_STRING, 8, C.uint(len("goclip")), C.CString("goclip"))
	i.win = selection_window

	//log.Printf("goclip: created")

	C.xcb_flush(i.dpy)
	wg.Done()

	for {
		//log.Printf("goclip: wait for event")
		ev := C.xcb_wait_for_event(i.dpy)
		if ev == nil {
			log.Printf("goclip: lost connection to X11")
			break
		}
		i.eventHandler(ev)
	}
}

func (i *internal) eventHandler(ev *C.xcb_generic_event_t) {
	defer C.xcb_flush(i.dpy)
	defer C.free(unsafe.Pointer(ev))

	evTyp := ev.response_type & 0x7f

	switch evTyp {
	case 0:
		// Error
		generr := (*C.xcb_generic_error_t)(unsafe.Pointer(ev))
		log.Printf("Got error %d from request %d:%d", generr.error_code, generr.major_code, generr.minor_code)
	case C.XCB_PROPERTY_NOTIFY: // 28
		//pEv := (*C.xcb_property_notify_event_t)(unsafe.Pointer(ev))
		//log.Printf("property notify=%+v", pEv)
		// notify=&{response_type:28 pad0:0 sequence:73 window:79691776 atom:485 time:3100655541 state:0 pad1:[0 0 0]}
		// ignore
	case C.XCB_SELECTION_REQUEST: // 30
		rEv := (*C.xcb_selection_request_event_t)(unsafe.Pointer(ev))
		i.handleSelectionRequest(rEv)

		// send completion notify
		notify := &C.xcb_selection_notify_event_t{
			response_type: C.XCB_SELECTION_NOTIFY,
			time:          rEv.time,
			requestor:     rEv.requestor,
			selection:     rEv.selection,
			target:        rEv.target,
			property:      rEv.property,
		}

		C.xcb_send_event(i.dpy, 0, rEv.requestor, C.XCB_EVENT_MASK_NO_EVENT, (*C.char)(unsafe.Pointer(notify)))
	case C.XCB_SELECTION_NOTIFY: // 31
		sEv := (*C.xcb_selection_notify_event_t)(unsafe.Pointer(ev))
		//log.Printf("got selection notify = %+v", sEv)
		// got selection notify = &{response_type:159 pad0:0 sequence:67 time:0 requestor:96468992 selection:1 target:485 property:485}

		switch sEv.property {
		case i.atom("TARGETS"):
			// regular event, read data & pass to triggerSel
			data := i.spawnData(sEv.selection, sEv.property)
			// triggerData happens in a separate thread
			go i.triggerData(data)
			return
		case i.atom("FOO"):
			//log.Printf("received FOO board event, sending to chan")
			b := i.linuxAtomToBoard(sEv.selection)
			select {
			case i.boardEvChan(b) <- evData{selection: sEv.selection, target: sEv.target, property: sEv.property}:
			default:
				log.Printf("WARNING: clipboard event dropped due to queue full for %s", b)
			}
		default:
			log.Printf("unhandled target: %v", sEv.target)
		}

	case i.query_ext.first_event + C.XCB_XFIXES_SELECTION_NOTIFY:
		fEv := (*C.xcb_xfixes_selection_notify_event_t)(unsafe.Pointer(ev))
		if fEv.owner == i.win {
			// do not worry about ourselves
			return
		}
		//log.Printf("xfixes event, new owner=%+v", fEv)
		// &{response_type:86 subtype:0 sequence:15 window:79691776 owner:58721633 selection:1 timestamp:3069285688 selection_timestamp:3069285672 pad0:[0 0 0 0 0 0 0 0]}
		C.xcb_convert_selection(i.dpy, i.win, fEv.selection, i.atom("TARGETS"), i.atom("TARGETS"), fEv.selection_timestamp) //C.XCB_CURRENT_TIME)

	default:
		log.Printf("goclip: got unknown event type %d", ev.response_type)
	}
}

func (i *internal) handleSelectionRequest(rEv *C.xcb_selection_request_event_t) {
	log.Printf("rev = %+v", rEv)
	// &{response_type:30 pad0:0 sequence:18 time:3095213350 owner:79691776 requestor:79691776 selection:477 target:485 property:485}

	board := i.linuxAtomToBoard(rEv.selection)

	i.copyValL.RLock()
	data, ok := i.copyVal[board]
	i.copyValL.RUnlock()

	if !ok {
		return // :(
	}

	tgt := i.resolveAtom(rEv.target)
	prop := i.resolveAtom(rEv.property)

	log.Printf("rev board=%s target=%s prop=%s", board, tgt, prop)

	switch tgt {
	case "TARGETS":
		var targets []C.xcb_atom_t
		targets = append(targets, i.atom("TARGETS"), i.atom("SAVE_TARGETS")) //, i.atom("MULTIPLE"))

		switch data.Type() {
		case Text:
			// add text targets
			targets = append(targets, i.atom("UTF8_STRING"), i.atom("COMPOUND_TEXT"), i.atom("TEXT"), i.atom("STRING"))
		}

		opts, err := data.GetAllFormats()
		if err != nil {
			log.Printf("failed to fetch formats: %s", err)
			break
		}
		for _, opt := range opts {
			m := opt.Mime()
			if m == "" {
				continue // ?
			}
			targets = append(targets, i.atom(m))
			ppos := strings.IndexByte(m, ';')
			if ppos != -1 {
				targets = append(targets, i.atom(m[:ppos]))
			}
		}

		C.xcb_change_property(i.dpy, C.XCB_PROP_MODE_REPLACE, rEv.requestor, rEv.property, C.XCB_ATOM_ATOM, 8*C.uint8_t(unsafe.Sizeof(C.xcb_atom_t(0))), C.uint32_t(len(targets)), unsafe.Pointer(&targets[0]))
		C.xcb_flush(i.dpy)
		return
	default:
		log.Printf("fetching value %s", tgt)
		switch tgt {
		case "TEXT", "UTF8_STRING", "STRING", "COMPOUND_TEXT":
			tgt = "text/plain"
		}
		buf, err := data.GetFormat(context.Background(), tgt)
		if err != nil {
			log.Printf("failed to fetch: %s", err)
			break
		}
		log.Printf("goclip: got %d bytes, setting", len(buf))
		C.xcb_change_property(i.dpy, C.XCB_PROP_MODE_REPLACE, rEv.requestor, rEv.property, rEv.target, 8, C.uint32_t(len(buf)), unsafe.Pointer(&buf[0]))
		C.xcb_flush(i.dpy)
		return
	}

	// if still here it means it failed
	C.xcb_change_property(i.dpy, C.XCB_PROP_MODE_REPLACE, rEv.requestor, rEv.property, C.XCB_ATOM_NONE, 0, 0, nil)
	C.xcb_flush(i.dpy)
}

func (i *internal) linuxAtomToBoard(sel C.xcb_atom_t) Board {
	switch sel {
	case i.atom("CLIPBOARD"):
		return Default
	case i.atom("PRIMARY"):
		return PrimarySelection
	case i.atom("SECONDARY"):
		return SecondarySelection
	default:
		return InvalidBoard
	}
}

func (i *internal) spawnData(sel, prop C.xcb_atom_t) Data {
	b := i.linuxAtomToBoard(sel)
	if prop == 0 {
		return emptyData{}
	}

	// prop==TARGETS (always)
	reply := C.xcb_get_property_reply(i.dpy, C.xcb_get_property(i.dpy, 1, i.win, prop, C.XCB_ATOM_ATOM, 0, 300), nil)
	defer C.free(unsafe.Pointer(reply))

	atomsPtr := C.xcb_get_property_value(reply)
	cnt := C.xcb_get_property_value_length(reply) / 4

	atoms := unsafe.Slice((*C.xcb_atom_t)(atomsPtr), int(cnt))

	var formats []DataOption

	//log.Printf("lookup atom response, got length=%d bytes", C.xcb_get_property_value_length(reply))
	for _, atomV := range atoms {
		f := i.resolveAtom(atomV)
		//log.Printf("%d: %s (%x)", c, f, atomV)
		formats = append(formats, atom{name: f, board: b, value: atomV})
	}

	return &StaticData{TargetBoard: b, Options: formats}
}

func (i *internal) triggerData(data Data) {
	for _, m := range i.mon {
		m.fire(data)
	}
}

func linuxBoardName(board Board) string {
	switch board {
	case Default:
		return "CLIPBOARD"
	case PrimarySelection:
		return "PRIMARY"
	case SecondarySelection:
		return "SECONDARY"
	default:
		return ""
	}
}
