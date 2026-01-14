[![GoDoc](https://godoc.org/github.com/KarpelesLab/goclip?status.svg)](https://godoc.org/github.com/KarpelesLab/goclip)
[![CI](https://github.com/KarpelesLab/goclip/actions/workflows/ci.yml/badge.svg)](https://github.com/KarpelesLab/goclip/actions/workflows/ci.yml)

# GoClip

Manipulate clipboard from Go, using system libraries.

Most clipboard implementations for Go out there rely on external programs to handle clipboard which limits what can be done quite a bit, only support text and do not have monitoring support.

GoClip aims to provide a cross platform API that can be used easily without compromise on what can be done.

## Features

* Easily read from or write to the clipboard
* Support for selection clipboard on X11
* Support for the following types of data:
  * Unicode Text
  * Images (returned as raw data in object with access methods to convert to `image.Image`)
  * File lists
* Notifications on clipboard contents updated (Monitor)
* All operations are done using the appropriate libs (no execution of external commands)
* On Windows acquiring ownership of the clipboard can take time. Contexts allows setting a timeout and a cancel method allowing for fine control on the process.

## Platform notes

* **Linux**: Uses X11 (xcb) for clipboard access. On Wayland, this requires XWayland to be running. Native Wayland clipboard support is not currently implemented.
* **macOS**: Uses native Cocoa APIs.
* **Windows**: Uses native Win32 APIs.

## Code samples

### Read from clipboard

```go
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	data, err := goclip.Paste(ctx)
	if err != nil {
		...
	}
	text, err := data.ToText(ctx)
	if err != nil {
		...
	}
	log.Printf("pasted text: %s", text)
```

Or

```go
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	data, _ := goclip.Paste(ctx)
	switch data.Type() { // data.Type() will return goclip.Invalid if no data
	case goclip.Image:
		img, err := data.ToImage(ctx) // converts data into image
	}
```

### Write to clipboard

```go
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	err := goclip.Copy(ctx, "Hello World") // copy text
	err := goclip.Copy(ctx, image.NewRGBA(...)) // copy image
	err := goclip.Copy(ctx, os.Open("...")) // file
```

### Monitoring

```go
	monitor, err := goclip.NewMonitor()
	if err != nil {
		...
	}
	monitor.Subscribe(func(d goclip.Data) error {
		...
	})
	...
	// call monitor.Poll() when gaining window focus, or on regular but slow-ish interval
```
