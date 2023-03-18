#include <libkern/OSAtomic.h>
#include <Cocoa/Cocoa.h>
#include <goclip_darwin.h>

#define Invalid 0;
#define	Text 1;
#define	Image 2;
#define	FileList 3;

ClipboardInternal *cocoaPbFactory() {
	ClipboardInternal *res = calloc(1, sizeof(ClipboardInternal));
	res->pb = [NSPasteboard generalPasteboard];
	res->cb = calloc(1, sizeof(ClipboardData));
	res->cbi = calloc(1, sizeof(ClipboardInformation));
	return res;
}

int cocoaPbChangeCount(ClipboardInternal *i) {
	return [i->pb changeCount];
}

// debug
const char* nsstring2cstring(void *s) {
	if (s == NULL) { return NULL; }

	NSString *nss = (NSString*)s;

	const char *cstr = [nss UTF8String];
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

void pasteWrite(ClipboardInternal *i) {
	if(pasteWriteItems == NULL) {
		return;
	}
	[i->pb writeObjects: pasteWriteItems];
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

void readClipboard(ClipboardInternal *i, ClipboardTypeFilter *filter) {
	i->cbi->count = [i->pb changeCount];

	for (id lastItem in [i->pb.pasteboardItems reverseObjectEnumerator]) {
		NSArray *types = [lastItem types];

		if (filter->text && (([types count] == 1 && [types containsObject:NSPasteboardTypeString]) || // plain text
		([types containsObject:NSPasteboardTypeHTML]) || // text with html tags
		([types containsObject:NSPasteboardTypeRTF]) || // Rich Text Format
		([types containsObject:NSPasteboardTypeRTFD])) // Rich Text Format Directory
		) {
			i->cbi->typeInt = CLIPBOARD_FORMAT_UTF8_TEXT;
			i->cbi->formatTypeInt = Text;
			extractData(i->cb, lastItem, NSPasteboardTypeString);
			return;
		}

		// image
		if (filter->image && [types containsObject:NSPasteboardTypeTIFF]) {
			i->cbi->formatTypeInt = CLIPBOARD_FORMAT_IMAGE_TIFF;
			i->cbi->typeInt = Image;

			extractData(i->cb, lastItem, NSPasteboardTypeTIFF);
			return;
		} else if (filter->image && [types containsObject:NSPasteboardTypePNG]) {
			i->cbi->formatTypeInt = CLIPBOARD_FORMAT_IMAGE_PNG;
			i->cbi->typeInt = Image;

			extractData(i->cb, lastItem, NSPasteboardTypePNG);
			return;
		}
	}
}

void readInformation(ClipboardInternal *i) {
	i->cbi->count = [i->pb changeCount];
	NSPasteboardItem *lastItem = [i->pb.pasteboardItems lastObject];
	NSArray *types = [lastItem types];

	if (([types count] == 1 && [types containsObject:NSPasteboardTypeString]) || // plain text
	([types containsObject:NSPasteboardTypeHTML]) || // text with html tags
	([types containsObject:NSPasteboardTypeRTF]) || // Rich Text Format
	([types containsObject:NSPasteboardTypeRTFD]) // Rich Text Format Directory
	) {
		i->cbi->typeInt = CLIPBOARD_FORMAT_UTF8_TEXT;
		i->cbi->formatTypeInt = Text;
		return;
	}

	// image
	if ([types containsObject:NSPasteboardTypeTIFF]) {
		i->cbi->formatTypeInt = CLIPBOARD_FORMAT_IMAGE_TIFF;
		i->cbi->typeInt = Image;
		return;
	} else if ([types containsObject:NSPasteboardTypePNG]) {
		i->cbi->formatTypeInt = CLIPBOARD_FORMAT_IMAGE_PNG;
		i->cbi->typeInt = Image;
		return;
	}
}
