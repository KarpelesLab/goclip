#include <libkern/OSAtomic.h>
#include <Cocoa/Cocoa.h>
#include <goclip_darwin.h>

#define Invalid 0;
#define	Text 1;
#define	Image 2;
#define	FileList 3;

NSPasteboard *cocoaPbFactory() {
	return [NSPasteboard generalPasteboard];
}

int cocoaPbChangeCount(NSPasteboard *pb) {
    return [pb changeCount];
}

// debug
const char* nsstring2cstring(NSString *s) {
    if (s == NULL) { return NULL; }

    const char *cstr = [s UTF8String];
    return cstr;
}


// return true if empty
bool cocoaPbIsEmpty(NSPasteboard* pb) {
    NSDictionary *options = [NSDictionary dictionary];
    NSArray *classes = [[NSArray alloc] initWithObjects:[NSString class], nil];
    bool has = [pb canReadObjectForClasses:classes options:options];
	[classes release];
    return !has;
}

// get all the regular text: NSString -- go-side deals with any further processing
void cocoaPbReadText(NSPasteboard* pb, void set(const char*, int)) {
    NSDictionary *options = [NSDictionary dictionary];
    NSArray *classes = [[NSArray alloc] initWithObjects:[NSString class], nil];
    NSArray *itms = [pb readObjectsForClasses:classes options:options];
    if (itms != nil) {
        int n = [itms count];
        int i;
        for (i=0; i<n; i++) {
            NSString* clip = [itms objectAtIndex: i];

            const char* utf8_clip = [clip UTF8String];
            set(utf8_clip, strlen(utf8_clip));
        }
    }
    [classes release];
    // [itms release]; // we do NOT own this!
}


static NSMutableArray *pasteWriteItems = NULL;

// add text to the list of items to paste
void pasteWriteAddText(char* data, int len) {
    NSString *ns_clip;
    bool ret;

    if(pasteWriteItems == NULL) {
        pasteWriteItems = [NSMutableArray array];
		 [pasteWriteItems retain];
    }

    ns_clip = [[NSString alloc] initWithBytes:data length:len encoding:NSUTF8StringEncoding];
    [pasteWriteItems addObject:ns_clip];
	[ns_clip release]; // pastewrite owns
}

void pasteWrite(NSPasteboard* pb) {
    if(pasteWriteItems == NULL) {
        return;
    }
    [pb writeObjects: pasteWriteItems];
	[pasteWriteItems release];
    pasteWriteItems = NULL;
}

void extractData(struct ClipboardData *cbData, NSPasteboardItem *item, NSPasteboardType type) {
    NSData *data = [[item dataForType:type] mutableCopy];

    NSUInteger len = [data length];
    cbData->dataLength = len;
    cbData->data = (Byte*)malloc(len);
    [data getBytes:cbData->data length:len];
}

void readClipboard(NSPasteboard *pb, struct ClipboardData *cbData, struct ClipboardInformation *cbInfo, ClipboardTypeFilter *filter) {
    cbInfo->count = [pb changeCount];

    for (id lastItem in [pb.pasteboardItems reverseObjectEnumerator]) {
        NSArray *types = [lastItem types];

        if (filter->text && (([types count] == 1 && [types containsObject:NSPasteboardTypeString]) || // plain text
        ([types containsObject:NSPasteboardTypeHTML]) || // text with html tags
        ([types containsObject:NSPasteboardTypeRTF]) || // Rich Text Format
        ([types containsObject:NSPasteboardTypeRTFD])) // Rich Text Format Directory
        ) {
            cbInfo->typeInt = CLIPBOARD_FORMAT_UTF8_TEXT;
            cbInfo->formatTypeInt = Text;
            extractData(cbData, lastItem, NSPasteboardTypeString);
            return;
        }

        // image
        if (filter->image && [types containsObject:NSPasteboardTypeTIFF]) {
            cbInfo->formatTypeInt = CLIPBOARD_FORMAT_IMAGE_TIFF;
            cbInfo->typeInt = Image;

            extractData(cbData, lastItem, NSPasteboardTypeTIFF);
            return;
        } else if (filter->image && [types containsObject:NSPasteboardTypePNG]) {
            cbInfo->formatTypeInt = CLIPBOARD_FORMAT_IMAGE_PNG;
            cbInfo->typeInt = Image;

            extractData(cbData, lastItem, NSPasteboardTypePNG);
            return;
        }
    }
}

void readInformation(NSPasteboard *pb, struct ClipboardInformation *cbInfo) {
    cbInfo->count = [pb changeCount];
    NSPasteboardItem *lastItem = [pb.pasteboardItems lastObject];
    NSArray *types = [lastItem types];

    if (([types count] == 1 && [types containsObject:NSPasteboardTypeString]) || // plain text
    ([types containsObject:NSPasteboardTypeHTML]) || // text with html tags
    ([types containsObject:NSPasteboardTypeRTF]) || // Rich Text Format
    ([types containsObject:NSPasteboardTypeRTFD]) // Rich Text Format Directory
    ) {
        cbInfo->typeInt = CLIPBOARD_FORMAT_UTF8_TEXT;
        cbInfo->formatTypeInt = Text;
        return;
    }

    // image
    if ([types containsObject:NSPasteboardTypeTIFF]) {
        cbInfo->formatTypeInt = CLIPBOARD_FORMAT_IMAGE_TIFF;
        cbInfo->typeInt = Image;
        return;
    } else if ([types containsObject:NSPasteboardTypePNG]) {
        cbInfo->formatTypeInt = CLIPBOARD_FORMAT_IMAGE_PNG;
        cbInfo->typeInt = Image;
        return;
    }
}
