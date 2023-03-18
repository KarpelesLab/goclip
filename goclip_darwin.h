#include "Cocoa/Cocoa.h"

#define CLIPBOARD_FORMAT_NONE 0
#define CLIPBOARD_FORMAT_UTF8_TEXT 1
#define CLIPBOARD_FORMAT_IMAGE_PNG 2
#define CLIPBOARD_FORMAT_IMAGE_BMP 3
#define CLIPBOARD_FORMAT_IMAGE_TIFF 4
#define CLIPBOARD_FORMAT_IMAGE_JPG 5

typedef struct ClipboardInformation {
	int typeInt;
	int formatTypeInt;
	int count;
} ClipboardInformation;

typedef struct ClipboardData {
	Byte* data;
	int dataLength;
} ClipboardData;

typedef struct ClipboardInternal {
	NSPasteboard *pb;
	ClipboardData *cb;
	ClipboardInformation *cbi;
} ClipboardInternal;

typedef struct ClipboardTypeFilter {
	bool text;
	bool image;
	bool files;
} ClipboardTypeFilter;

ClipboardInternal *cocoaPbFactory();
int cocoaPbChangeCount(ClipboardInternal *sub);
void pasteWriteAddText(char* data, int len);
void pasteWrite(ClipboardInternal *sub);

void readClipboard(ClipboardInternal *i, ClipboardTypeFilter *filter);
void readInformation(ClipboardInternal *i);

const char* nsstring2cstring(void *str);
