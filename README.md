[![GoDoc](https://godoc.org/github.com/KarpelesLab/goclip?status.svg)](https://godoc.org/github.com/KarpelesLab/goclip)

# GoClip

**WORK IN PROGRESS** This is not ready for use yet.

Manipulate clipboard from Go, using system libraries.

Most clipboard implementations for Go out there rely on external programs to handle clipboard which limits what can be done quite a bit, only support text and do not have monitoring support.

GoClip aims to provide a cross platform API that can be used easily without compromise on what can be done.

## Target features

* Easily read from or write to the clipboard
* Support for selection clipboard on X11
* Support for the following types of data:
  * Unicode Text
  * Images (returned as raw data in object with access methods to convert to `image.Image`)
  * File lists
* Notifications on clipboard contents updated (Monitor)
* All operations are done using the appropriate libs (no execution of external commands)
* On Windows acquiring ownership of the clipboard can take time. Contexts allows setting a timeout and a cancel method allowing for fine control on the process.

## Code samples

**Warning**: this will not work. This code is only there to illustrate the goal for this project.

### Read from clipboard

```go
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	data, err := goclip.Paste(ctx, goclip.Text)
	if err != nil {
		...
	}
	log.Printf("pasted text: %s", data.String())
```

Or

```go
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	data, _ := goclip.Paste(ctx, goclip.Text, goclip.Image, goclip.FileList)
	switch data.Type() { // data.Type() will return goclip.Invalid if no data
	case goclip.Image:
		img, err := data.Image() // converts data into image
	}
```

### Write to clipboard

```go
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	err := goclip.Copy(ctx, "Hello world") // copy text
	err := goclip.Copy(ctx, image.NewRGBA(...)) // copy image
	err := goclip.Copy(ctx, os.Open("...")) // file
```

### Monitoring

```go
	monitor := goclip.NewMonitor()
	go func() {
		for _, ev := range monitor.C {
			...
		}
	}()
	...
	// call monitor.Poll() when gaining window focus, or on regular interval
```
