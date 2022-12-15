#include "Cocoa/Cocoa.h"

#define CLIPBOARD_FORMAT_NONE 0;
#define CLIPBOARD_FORMAT_UTF8_TEXT 1;
#define CLIPBOARD_FORMAT_IMAGE_PNG 2;
#define CLIPBOARD_FORMAT_IMAGE_BMP 3;
#define CLIPBOARD_FORMAT_IMAGE_TIFF 4;
#define CLIPBOARD_FORMAT_IMAGE_JPG 5;

NSPasteboard *cocoaPbFactory();
int cocoaPbChangeCount(NSPasteboard *pb);
void pasteWriteAddText(char* data, int len);
void pasteWrite(NSPasteboard* pb);

typedef struct ClipboardInformation {
	int typeInt;
	int formatTypeInt;
	int count;
} ClipboardInformation;

typedef struct ClipboardData {
	Byte* data;
	int dataLength;
} ClipboardData;

typedef struct ClipboardTypeFilter {
    bool text;
    bool image;
    bool files;
} ClipboardTypeFilter;

void readClipboard(NSPasteboard *pb, ClipboardData* cbData, ClipboardInformation *cbInfo, ClipboardTypeFilter *filter);
void readInformation(NSPasteboard *pb, ClipboardInformation *cbInfo);

const char* nsstring2cstring(NSString *str);
